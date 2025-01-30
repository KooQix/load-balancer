package main

// Dumb server to handle requests (main will redirect requests to this server)

import (
	"log"
	"net/http"
)


func Server() {
	mux := http.NewServeMux()

	// Define a map of valid endpoints and their handlers
    endpoints := map[string]http.HandlerFunc{
        "/": func(w http.ResponseWriter, r *http.Request) {
            // Only respond with "Hello, World!" if the path is exactly "/"
            if r.URL.Path != "/" {
                w.WriteHeader(http.StatusNotFound)
                w.Write([]byte("404 - Not Found"))
                return
            }
            w.Write([]byte("Hello, World!"))
        },
        "/test": func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Hello, Test!"))
        },
    }

    // Register the endpoints
    for path, handler := range endpoints {
        mux.HandleFunc(path, handler)
    }


	srv := &http.Server{
		Addr: ":8081",
		Handler: mux,
	}

	log.Println("Server listening on :8081")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
