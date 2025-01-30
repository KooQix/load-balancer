package server

import (
	"load-balancer/src/queue"
	"math"
)

// findGCD finds the Greatest Common Divisor of two numbers
func findGCD(a, b int) int {
    for b != 0 {
        a, b = b, a%b
    }
    return a
}

// findGCDOfArray finds the GCD of an array of numbers
func findGCDOfArray(servers []*Node) int {
    result := servers[0].RateLimit
    for i := 1; i < len(servers); i++ {
        result = findGCD(result, servers[i].RateLimit)
    }
    return result
}

func fillQueue(queue *queue.RingQueue[*Node], servers []*Node) {
	// Find minimum rate limit
    minRate := servers[0].RateLimit
    for _, server := range servers {
        if server.RateLimit < minRate {
            minRate = server.RateLimit
        }
    }

    // Calculate ratios and find GCD to optimize queue size
    ratios := make([]int, len(servers))
	gcdArray := findGCDOfArray(servers)
    for i, server := range servers {
        ratios[i] = server.RateLimit / gcdArray
    }

    // Calculate total weight and create queue
    totalWeight := 0
    for _, ratio := range ratios {
        totalWeight += ratio
    }

    weights := make([]float64, len(servers))
    for i := range weights {
        weights[i] = 0.0
    }

    // Fill the queue using the modified weighted round-robin algorithm
    for queue.Length() < totalWeight {
        maxWeight := math.Inf(-1)
        selectedServer := -1

        // Find the server with the highest weight
        for i := range servers {
            weights[i] += float64(ratios[i])
            if weights[i] > maxWeight {
                maxWeight = weights[i]
                selectedServer = i
            }
        }

        // Add selected server to queue and decrease its weight
		queue.Add(servers[selectedServer])
        weights[selectedServer] -= float64(totalWeight)
    }
}

// createWeightedQueue creates an optimized weighted round-robin queue (eg based on rate limits, create a queue with the optimal size. The number of occurrences of each server in the queue is proportional to its rate limit)
// If all the servers have the same rate limit, the queue will be a simple round-robin queue
func createWeightedQueue(servers []*Node) *queue.RingQueue[*Node] {
	queue := queue.New[*Node]()

	// If no servers are provided, return an empty queue
    if len(servers) == 0 {
        return queue
    }

    fillQueue(queue, servers)

    return queue
}