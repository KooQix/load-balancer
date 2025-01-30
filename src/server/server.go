package server

import (
	"load-balancer/src/queue"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	// "github.com/go-redis/redis"
	"golang.org/x/time/rate"
)

// Configuration struct for the application
type Config struct {
	PostgresURL string
	// RedisURL    string
	CacheSize   int
	CacheTTL    time.Duration
}

// RPCServer represents a server record from the database
type RPCServer struct {
	ID         int       `json:"id"` // ID is the primary key
	URL        string    `json:"url"`
	RateLimit  int       `json:"rate_limit"`
	BurstLimit int       `json:"burst_limit"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Node extends RPCServer with runtime information
type Node struct {
	*RPCServer
	limiter *rate.Limiter
}


// ServerManager handles server management and caching
type ServerManager struct {
	db          *sql.DB
	// redis       *redis.Client
	cache       *queue.RingQueue[*Node]
	cacheMutex  sync.RWMutex
	cacheSize   int
	cacheTTL    time.Duration
	refreshTick time.Duration
}

var healthConfig HealthConfig

func init() {
	healthConfig = loadHealthConfig()
}

// NewServerManager creates a new server manager instance
func NewServerManager(config Config) (*ServerManager, error) {
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %v", err)
	}

	// // Connect to Redis
	// opt, err := redis.ParseURL(config.RedisURL)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse redis URL: %v", err)
	// }
	
	// rdb := redis.NewClient(opt)
	
	// // Test Redis connection
	// if _, err := rdb.Ping(context.Background()).Result(); err != nil {
	// 	return nil, fmt.Errorf("failed to connect to redis: %v", err)
	// }


	sm := &ServerManager{
		db:          db,
		// redis:       rdb,
		cache:       nil,
		cacheSize:   config.CacheSize,
		cacheTTL:    config.CacheTTL,
		refreshTick: 15 * time.Minute, // Refresh cache every 15 minutes
	}

	// Initialize the cache
	ctx := context.Background()
	activeServers, err := sm.getActiveServers(ctx)
	if err != nil {
        log.Fatalf("failed to get active servers: %v", err)
    }

	nodes := make([]*Node, 0, len(activeServers))
	for _, server := range activeServers {
		nodes = append(nodes, &Node{
            RPCServer: server,
            limiter:   rate.NewLimiter(rate.Limit(server.RateLimit), server.BurstLimit),
        })
	}

	sm.cache = createWeightedQueue(nodes)

	// Start cache refresh routine
	go sm.startCacheRefresh()

	// Starts the health check routine
	go sm.startHealthCheck()

	return sm, nil
}

// startCacheRefresh periodically refreshes the cache
func (sm *ServerManager) startCacheRefresh() {
	ticker := time.NewTicker(sm.refreshTick)
	for range ticker.C {
		if err := sm.refreshCache(); err != nil {
			log.Printf("Error refreshing cache: %v", err)
		}
	}
}

// refreshCache updates the cache with fresh data from the database
func (sm *ServerManager) refreshCache() error {
	ctx := context.Background()
	
	// Get all active servers from database
	servers, err := sm.getActiveServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh cache: %v", err)
	}


	// Only refresh cache if the number of servers have changed (other updates are made on cache AND database, so the only thing that can change is the number of servers)
	if len(servers) == sm.cache.Length() {
		return nil
	}

	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	nodes := make([]*Node, 0, len(servers))

	// Update cache with fresh data
	for _, server := range servers {
		node := &Node{
			RPCServer: server,
			limiter:   rate.NewLimiter(rate.Limit(server.RateLimit), server.BurstLimit),
		}
		nodes = append(nodes, node)

		// // Update Redis cache
		// serverJSON, err := json.Marshal(server)
		// if err != nil {
		// 	log.Printf("Error marshaling server %d: %v", server.ID, err)
		// 	continue
		// }

		// err = sm.redis.Set(ctx, fmt.Sprintf("server:%d", server.ID), serverJSON, sm.cacheTTL).Err()
		// if err != nil {
		// 	log.Printf("Error updating Redis cache for server %d: %v", server.ID, err)
		// }
	}

	sm.cache = createWeightedQueue(nodes)

	return nil
}

// GetActiveServers retrieves all active servers from the database
func (sm *ServerManager) getActiveServers(ctx context.Context) ([]*RPCServer, error) {
	query := `
		SELECT id, url, rate_limit, burst_limit, is_active, created_at, updated_at
		FROM servers
		WHERE is_active = true
		ORDER BY id ASC
	`

	rows, err := sm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active servers: %v", err)
	}
	defer rows.Close()

	var servers []*RPCServer

	// Add an index to the servers (to ensure deterministic order, since the database doesn't guarantee it)
	for rows.Next() {
		server := &RPCServer{}
		err := rows.Scan(
			&server.ID,
			&server.URL,
			&server.RateLimit,
			&server.BurstLimit,
			&server.IsActive,
			&server.CreatedAt,
			&server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan server row: %v", err)
		}

		servers = append(servers, server)
	}

	return servers, nil
}


func (sm *ServerManager) getNextNode() *Node {
	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	if sm.cache.Length() == 0 {
		return nil
	}

	node := sm.cache.Next()
	for sm.cache.Length() > 0 && (node == nil || !node.IsActive) {
		// Remove all inactive nodes starting from current
		// The current is the next node, since "Next" returns the current and moves to the next
		for sm.cache.Length() > 0 && !sm.cache.Current().IsActive {
			sm.cache.Remove()
		}

		if sm.cache.Length() == 0 {
			return nil
		}

		node = sm.cache.Next()
	}

	return node

}


func (sm *ServerManager) setServerActive(server *RPCServer, active bool) {
	
	// Update in-memory cache (just have to set the value of the pointer, Node refers to the RPCServer pointer, and the queue itself refers to the Node pointers), so updating the cache is easy.
	// When a server has been set to inactive, it will be removed from the cache the next time it is accessed
	sm.cacheMutex.Lock()
	server.IsActive = active
	sm.cacheMutex.Unlock()


	// // Update Redis cache
	// serverJSON, err := json.Marshal(node.RPCServer)
	// if err != nil {
	// 	log.Printf("Error marshaling server %d: %v", node.ID, err)
	// 	return
	// }

	// err = sm.redis.Set(ctx, fmt.Sprintf("server:%d", node.ID), serverJSON, sm.cacheTTL).Err()

	// if err != nil {
	// 	log.Printf("Error updating Redis cache for server %d: %v", node.ID, err)
	// }

	// Set database
	query := `
		UPDATE servers
		SET is_active = $1
		WHERE id = $2
	`

	_, err := sm.db.Exec(query, active, server.ID)

	if err != nil {
		log.Printf("Error updating server %d in database: %v", server.ID, err)
		return
	}
}


func (server *RPCServer) isHealthy() bool {
	// Check if the server is healthy
	// Since the servers are RPC servers, we can send a simple request to check if they are healthy

	jsonBody, err := json.Marshal(healthConfig.HealthCheck.Request.Body)

	if err != nil {
		log.Printf("Error marshaling health check request body: %v", err)
        return false
	}
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(healthConfig.HealthCheck.Request.Method, server.URL, bodyReader)
	if err != nil {
		log.Printf("Error creating health check request: %v", err)
		return false
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending health check request: %v", err)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Server %d is unhealthy: %v", server.ID, resp.Status)
		return false
	}

	return true
}


func (sm *ServerManager) startHealthCheck() {
	var duration time.Duration

	switch healthConfig.HealthCheck.Interval.Unit {
	case "hour":
		duration = time.Hour
	case "minute":
		duration = time.Minute
	case "second":
		duration = time.Second
	}

	duration *= time.Duration(healthConfig.HealthCheck.Interval.Value)

	// Run health check every 24hours
	ticker := time.NewTicker(duration)
    for range ticker.C {
        sm.healthCheck()
    }
}

// healthCheck periodically checks the health of inactive servers (every 24 hours) meaning it checks if the server responds to requests, and updates the database accordingly
func (sm *ServerManager) healthCheck() {
	log.Println("Starting health check routine")
	
	// Get all the inactive servers
	query := `
		SELECT * FROM servers WHERE is_active = false`

	rows, err := sm.db.Query(query)
	if err != nil {
		log.Printf("Error querying inactive servers: %v", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		server := &RPCServer{}
		err := rows.Scan(
			&server.ID,
			&server.URL,
			&server.RateLimit,
			&server.BurstLimit,
			&server.IsActive,
			&server.CreatedAt,
			&server.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning inactive server: %v", err)
			continue
		}

		// Check if the server is healthy, if so, set it as active
		if server.isHealthy() {
			sm.setServerActive(server, true)
		}
	}
}