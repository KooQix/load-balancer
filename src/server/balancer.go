package server

import (
	"load-balancer/src/prometheus"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var MAX_NODES_TRIED int = 5

// Balancer struct with server manager
type Balancer struct {
	ServerManager *ServerManager
	ReverseProxy bool
}

// handleStats returns statistics about the servers
func (b *Balancer) HandleStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	servers, err := b.ServerManager.getActiveServers(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get servers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_servers": len(servers),
		"servers":        servers,
	})
}


func (b *Balancer) makeRequest(ctx context.Context, url *url.URL, w http.ResponseWriter, r *http.Request, node *Node) bool {
	// Start timing the request
	start := time.Now()

	forwardReq, err := http.NewRequestWithContext(ctx, r.Method, url.String(), r.Body)
	forwardReq.RemoteAddr = r.RemoteAddr
	
	if err != nil {
		http.Error(w, "Failed to create forward request", http.StatusInternalServerError)
		return false
	}

	// Copy the headers from the original request
	forwardReq.Header = r.Header.Clone()
	forwardReq.ContentLength = r.ContentLength


	// Make the request
	resp, err := http.DefaultClient.Do(forwardReq)
	if err != nil {
		// Node might be down or other error
		// Increment error counter for this node
		prometheus.NodeErrors.WithLabelValues(node.URL).Inc()
		log.Printf("Node %s request error: %v\n", node.URL, err)
		return false
	}
	defer resp.Body.Close()

	// Record latency
	prometheus.RequestLatency.WithLabelValues(node.URL).Observe(time.Since(start).Seconds())

	// Write the response code and headers back to the client
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// If received a forbidden status code, then set the server as inactive. A goroutine will check the server status and set it as active again if it is up.
	if resp.StatusCode == http.StatusForbidden {
		go b.ServerManager.setServerActive(node.RPCServer, false)
	}

	// Stream the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v\n", err)
		return false
	}

	return true
}

func (b *Balancer) makeRedirect(url *url.URL, w http.ResponseWriter, r *http.Request, node *Node) {
	// Start timing the request
	start := time.Now()
	// Balancer set as redirect -> Redirect the client to the node's URL
	http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
	
	// Record latency
	prometheus.RequestLatency.WithLabelValues(node.URL).Observe(time.Since(start).Seconds())
}

// handleRequest receives the client request, picks an available node,
// and then proxifies the request to that node (if not rate-limited).
func (b *Balancer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()


	// Increment total requests counter
	prometheus.TotalRequests.Inc()

	// Try each node in a loop
	i := 0
	for {
		// Get next node from the server manager (round-robin), and get next if rate-limited
		node := b.ServerManager.getNextNode()
		if node == nil {
			http.Error(w, "No active RPC nodes available", http.StatusServiceUnavailable)
			return
		}
		
		if i >= MAX_NODES_TRIED {
			// If we couldn't find any node that can handle the request right now
			// (e.g., all are rate-limited), return an HTTP 429 or 503
			http.Error(w, "All RPC nodes are busy at the moment", http.StatusTooManyRequests)
			prometheus.TotalRateLimitHits.Inc()
			return
		}
		i++
	
		if node.limiter.Allow() {
			// Increment per-node request counter
			prometheus.PerNodeRequests.WithLabelValues(node.URL).Inc()

			// Parse the base redirect URL
			baseURL, err := url.Parse(node.URL)

			if err != nil {
				http.Error(w, "Failed to parse redirect URL", http.StatusInternalServerError)
				continue
			}

			// Combine the base URL with the original request path
			baseURL.Path, _ = url.JoinPath(baseURL.Path, r.URL.Path)
			baseURL.RawQuery = r.URL.RawQuery

			
			if !b.ReverseProxy {
				b.makeRedirect(baseURL, w, r, node)	
				return
			}

			// Proxy this request to node.URL
			if b.makeRequest(ctx, baseURL, w, r, node) {
				return
			}

		} else {
			// Increment rate limit hit counter for this node
			prometheus.RateLimitHits.WithLabelValues(node.URL).Inc()
		}
	}
}

func (b *Balancer) HandleSetInactiveNode(w http.ResponseWriter, r *http.Request) {
	nodeIDStr := r.URL.Query().Get("node_id")

	if nodeIDStr == "" {
		http.Error(w, "Missing node_id query parameter", http.StatusBadRequest)
		return
	}

	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil {
		http.Error(w, "Invalid node_id query parameter", http.StatusBadRequest)
		return
	}

	node := b.ServerManager.cache.HeadElement()
	for node.Value.ID != nodeID  {
		node = node.Next
	}
	if node == nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}
	b.ServerManager.setServerActive(node.Value.RPCServer, false)
	w.WriteHeader(http.StatusOK)
}