// main.go
package main

import (
	"load-balancer/src/server"
	"log"
	"net/http"
	"os"

	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	postgresURL string
	API_KEY string
	ADMIN_API_KEY string
	serveAsProxy bool
)

func init() {
	args := os.Args[1:]
	var envFile string

	if len(args) > 0 {
		devMode := args[0] == "dev"
		if devMode {
			envFile = ".env.development"
		} else {
			envFile = ".env"
		}
	} else {
		envFile = ".env"
	}

	log.Printf("Loading environment variables from %s\n", envFile)
	
	// Load environment variables from .env file
	var err error
	if err = godotenv.Load(envFile); err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	postgresURL = os.Getenv("POSTGRES_URL")

	if postgresURL == "" {
		log.Fatalln("POSTGRES_URL environment variable is required")
	}

	API_KEY = os.Getenv("API_KEY")
	ADMIN_API_KEY = os.Getenv("ADMIN_API_KEY")

	if API_KEY == "" {
		log.Fatalln("API_KEY environment variable is required")
	}

	if ADMIN_API_KEY == "" {
		log.Fatalln("ADMIN_API_KEY environment variable is required")
	}

	serveAsProxyStr := os.Getenv("BALANCER_TYPE")
	if serveAsProxyStr != "proxy" && serveAsProxyStr != "redirect"  {
		log.Fatalf("Invalid value for BALANCER_TYPE: %s. Must be 'proxy' or 'redirect'", serveAsProxyStr)
	}

	serveAsProxy = serveAsProxyStr == "proxy"

	log.Printf("Using %s mode\n", serveAsProxyStr)
}


// Main application code
func main() {
	config := server.Config{
		PostgresURL: postgresURL,
		// RedisURL:    os.Getenv("REDIS_URL"),
		CacheSize:   100,                    // Cache up to 100 servers
		CacheTTL:    15 * time.Minute,       // Cache TTL of 15 minutes
	}

	serverManager, err := server.NewServerManager(config)
	if err != nil {
		log.Fatalf("Failed to create server manager: %v", err)
	}

	// Initialize the balancer with servers from the database
	balancer := &server.Balancer{
		ServerManager: serverManager,
		ReverseProxy: serveAsProxy,
	}

	// Create a new mux server (handles panic recovery and auth)
	mux := &MuxServer{http.NewServeMux()}
	
	// Main endpoint for handling requests
	mux.HandleAuthFunc("/", balancer.HandleRequest)

	// Add an endpoint to set a server as inactive (if received a Forbidden error after a redirect)
	mux.HandleAuthFunc("/inactive-server", balancer.HandleSetInactiveNode)
	
	// Prometheus metrics endpoint
	mux.HandleAuthAdminFunc("/metrics", promhttp.Handler().ServeHTTP)

	// Add an endpoint to get server stats
	mux.HandleAuthAdminFunc("/stats", balancer.HandleStats)

	srv := &http.Server{
		Addr:    ":8000",
		Handler: mux,
		// ReadTimeout:    15 * time.Second,
        // WriteTimeout:   15 * time.Second,
        // IdleTimeout:    60 * time.Second,
        // MaxHeaderBytes: 1 << 20,
	}

	log.Println("Load Balancer listening on :8000")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
