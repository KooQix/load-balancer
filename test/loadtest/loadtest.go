package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type Statistics struct {
	mutex        sync.Mutex
	statusCounts map[int]int
	errors       int
}

func (s *Statistics) addStatus(status int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.statusCounts[status]++
}

func (s *Statistics) addError() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.errors++
}

func makeRequest(client *http.Client, url string, stats *Statistics, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		stats.addError()
		return
	}

	defer resp.Body.Close()

	// Read and discard the body to properly close the connection
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		stats.addError()
		return
	}

	stats.addStatus(resp.StatusCode)
}

func requestGroup(url string, concurrentRequests int, stats *Statistics) {
	var wg sync.WaitGroup
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	wg.Add(concurrentRequests)
	for i := 0; i < concurrentRequests; i++ {
		go makeRequest(client, url, stats, &wg)
	}
	wg.Wait()
}


func validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}
	if !parsedURL.IsAbs() {
		return fmt.Errorf("URL must be absolute (include http:// or https://)")
	}
	return nil
}

func validateFlags(totalReqs, concurrentReqs, delaySeconds int) error {
	if totalReqs <= 0 {
		return fmt.Errorf("total requests must be greater than 0")
	}
	if concurrentReqs <= 0 {
		return fmt.Errorf("concurrent requests must be greater than 0")
	}
	if concurrentReqs > totalReqs {
		return fmt.Errorf("concurrent requests cannot be greater than total requests")
	}
	if delaySeconds < 0 {
		return fmt.Errorf("delay seconds cannot be negative")
	}
	return nil
}

func main() {
	// Define command line flags
	urlFlag := flag.String("url", "http://localhost:8000/", "Target URL for load testing")
	totalReqs := flag.Int("n", 2000, "Total number of requests")
	concurrentReqs := flag.Int("c", 65, "Number of concurrent requests")
	delaySeconds := flag.Int("d", 1, "Delay between request groups in seconds (not applied if processing the group takes longer)")

	// Parse flags
	flag.Parse()

	// Validate URL
	if err := validateURL(*urlFlag); err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	// Validate numeric parameters
	if err := validateFlags(*totalReqs, *concurrentReqs, *delaySeconds); err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}

	url := *urlFlag
	totalRequests := *totalReqs
	concurrentRequests := *concurrentReqs
	delaySecondsDuration := time.Duration(*delaySeconds) * time.Second

	log.Printf("Starting load test on %s with %d total requests, %d concurrent requests, and %d second delay\n", url, totalRequests, concurrentRequests, *delaySeconds)

	// Calculate number of groups
	numGroups := (totalRequests + concurrentRequests - 1) / concurrentRequests

	stats := &Statistics{
		statusCounts: make(map[int]int),
	}

	startTime := time.Now()

	for group := 0; group < numGroups; group++ {
		groupStart := time.Now()

		// Calculate remaining requests for last group
		currentBatch := concurrentRequests
		if (group+1)*concurrentRequests > totalRequests {
			currentBatch = totalRequests - group*concurrentRequests
		}

		fmt.Printf("\n---------- Group %d/%d - %s ----------\n", group+1, numGroups, time.Now().Format("15:04:05"))
		fmt.Printf("Sending %d concurrent requests...\n", currentBatch)

		requestGroup(url, currentBatch, stats)

		// Print group statistics
		fmt.Printf("Group completed. Status codes: %v, Errors: %d\n", stats.statusCounts, stats.errors)

		// Calculate time spent and delay accordingly
		groupDuration := time.Since(groupStart)
		if group < numGroups-1 { // Don't delay after last group
			sleepTime := delaySecondsDuration - groupDuration
			if sleepTime > 0 {
				fmt.Printf("Waiting %.2fs before next group...\n", sleepTime.Seconds())
				time.Sleep(sleepTime)
			}
		}
	}

	// Print final statistics
	totalTime := time.Since(startTime)
	requestsPerSecond := float64(totalRequests) / totalTime.Seconds()

	fmt.Printf("\nFinal Statistics:\n")
	fmt.Printf("Total Requests: %d\n", totalRequests)
	fmt.Printf("Total Time: %.2f seconds\n", totalTime.Seconds())
	fmt.Printf("Requests per second: %.2f\n", requestsPerSecond)
	fmt.Printf("Status Code Distribution: %v\n", stats.statusCounts)
	fmt.Printf("Total Errors: %d\n", stats.errors)
}