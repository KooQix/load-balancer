package main

import (
	"fmt"
	"math"

	"load-balancer/src/queue"
)

type Server struct {
    ID        int
    RateLimit int
}



// findGCD finds the Greatest Common Divisor of two numbers
func findGCD(a, b int) int {
    for b != 0 {
        a, b = b, a%b
    }
    return a
}

// findGCDOfArray finds the GCD of an array of numbers
func findGCDOfArray(servers []*Server) int {
    result := servers[0].RateLimit
    for i := 1; i < len(servers); i++ {
        result = findGCD(result, servers[i].RateLimit)
    }
    return result
}

// createWeightedQueue creates an optimized weighted round-robin queue
func createWeightedQueue(servers []*Server) *queue.RingQueue[*Server] {
	// Create a new Ring Queue (if the queue size is relatively stable and you want maximum performance and memory efficiency.) This implementation is not thread-safe, so it's possible to fetch and add elements concurrently.
	// If the queue size is not stable, you can use a slice or a linked list.
	queue := queue.New[*Server]()

	// If no servers are provided, return an empty queue
    if len(servers) == 0 {
        return queue
    }

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

    return queue
}

func printQueue(queue *queue.RingQueue[*Server]) {
	node := queue.Next()
	for i := 0; i < queue.Length(); i++ {
		fmt.Printf("%v,", node.ID)
		node = queue.Next()
	}
	
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\nLength: %v, Current: %v, Head: %v\n", queue.Length(), nil, nil)
		}
	}()
	fmt.Printf("\nLength: %v, Current: %v, Head: %v\n", queue.Length(), queue.Current() , queue.Head())
}

func printServers(servers []*Server) {
	for _, server := range servers {
		fmt.Printf("{Server %d: Rate Limit %d},", server.ID, server.RateLimit)

	}
	fmt.Println()
}





func main() {
    // Example usage
	fmt.Println("Weighted Round Robin Queue Generation")
    servers := []*Server{
        {ID: 1, RateLimit: 15},
        {ID: 2, RateLimit: 20},
        {ID: 3, RateLimit: 30},
		{ID: 4, RateLimit: 45},
        {ID: 5, RateLimit: 15},
        {ID: 6, RateLimit: 25},
    }

    // queue := createWeightedQueue(servers)
	queue := &queue.RingQueue[*Server]{}
	for _, server := range servers {
		queue.Add(server)
		printQueue(queue)
	}
    // printServers(servers)

	fmt.Println()
	for i := 0; i < 2*len(servers); i++ {
		fmt.Printf("Value: %v, Next: %v\n", queue.Next(), queue.Current())
	}

	fmt.Println("\nRemove")

	for i := 0; i < len(servers); i++ {
		queue.Remove()
		printQueue(queue)
	}


	// fmt.Println("\nRound Robin Queue Generation")
	// servers = []*Server{
    //     {ID: 1, RateLimit: 15},
    //     {ID: 2, RateLimit: 15},
    //     {ID: 3, RateLimit: 15},
    // }

    // queue = createWeightedQueue(servers)
	// printServers(servers)
	// printQueue(queue)
}