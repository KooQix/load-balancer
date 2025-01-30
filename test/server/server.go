package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"sync"
)

// Create dumb servers to handle requests
// Create several servers to handle requests

func startServer(port int) {
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
		Addr: ":" + strconv.Itoa(port),
		Handler: mux,
	}

	log.Println("Server listening on :" + strconv.Itoa(port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}


}

func main() {
	// Get the arguments passed to the program
	numServs := flag.Int("num-servers", 1, "Number of servers to create")

	startPort := 8080

	flag.Parse()

	if *numServs < 1 {
		log.Fatalf("Number of servers must be at least 1")
	}

	if *numServs > 8 {
		log.Fatalf("Number of servers cannot exceed 8")
	}

	log.Printf("Creating %d servers\n", *numServs)

	wg := sync.WaitGroup{}

	for i := 0; i < *numServs; i++ {
		wg.Add(1)
		go startServer(startPort + i)
	}

	wg.Wait()
}