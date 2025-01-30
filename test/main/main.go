package main

// Test file to redirect requests to another server

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

var REDIRECT_URL string = "http://localhost:8081"

func main() {
	mux := http.NewServeMux()

	// Single endpoint for handling requests
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Parse the base redirect URL
		baseURL, err := url.Parse(REDIRECT_URL)
		if err != nil {
			http.Error(w, "Failed to parse redirect URL", http.StatusInternalServerError)
			return
		}

		// Combine the base URL with the original request path
		baseURL.Path = r.URL.Path

		// Preserve query parameters
        baseURL.RawQuery = r.URL.RawQuery

		// Create the forwarded request with the complete URL
		forwardReq, err := http.NewRequestWithContext(ctx, r.Method, baseURL.String(), r.Body)
		if err != nil {
			http.Error(w, "Failed to forward request", http.StatusInternalServerError)
			return
		}

		// Copy the headers from the original request
		forwardReq.Header = r.Header.Clone()

		// Make the request
		resp, err := http.DefaultClient.Do(forwardReq)
		if err != nil {
			http.Error(w, "Failed to forward request", http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		// Write the response code and headers back to the client
		for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
		}
		w.WriteHeader(resp.StatusCode)

		//  Stream the response body
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("Failed to stream response body: %v\n", err)
			
		}
	})

	// Run dump server (server.go) so that we can forward requests to it
	go Server()

	srv := &http.Server{
		Addr: ":8000",
		Handler: mux,
	}

	// Start server
	log.Println("Redirect Server listening on :8000")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

}