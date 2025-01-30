# Load balancer

Load balancer using Go, Prometheus, and Grafana. The load balancer uses a round-robin algorithm to distribute the requests among the servers. It implements request limiting to ensure the number of requests per server is not reached. It also collects metrics and exposes them to Prometheus.

To distribute the requests, the load balancer uses a list of servers stored in a database and creates an optimized weighted round-robin queue (eg based on rate limits, creates a queue with the optimal size. The number of occurrences of each server in the queue is proportional to its rate limit). If all the servers have the same rate limit, the queue will be a simple round-robin queue.

It does not route to inactive servers, but performs health check to verify if the server is active again.

# Architecture

The project is composed of the following services:

-   Load Balancer: Go application that distributes the requests among the servers.
-   Prometheus: Monitoring tool that collects metrics from the load balancer.
-   Grafana: Visualization tool that displays the metrics collected by Prometheus.
-   Database: Database to store the servers and their rate limits.
-   Nginx: Reverse proxy to route the requests to the load balancer.

# Requirements

-   docker
-   docker-compose

# Configuration

1. Copy the environment file and set the environment variables:

```bash
cp .env.example .env
```

2. Set the `config.json` file, which contains information to perform health checks.
   Example of `config.json`:

```json
{
	// Health check configuration
	// This checks if the inactive servers are active again, every 24 hours
	"healthCheck": {
		"interval": {
			"unit": "hour",
			"value": 24
		},
		"request": {
			// When a server is set to inactive, will try this request to check if it is active again (receive 200 status code)
			"method": "POST",
			"body": {
				"jsonrpc": "2.0",
				"id": 1,
				"method": "getLatestBlockhash",
				"params": [
					{
						"commitment": "processed"
					}
				]
			}
		}
	}
}
```

3. Update the ports in the `docker-compose.yml` file if necessary.

4. Update the `prometheus/prometheus.yml` file to include the API key to access the Prometheus metrics.

5. Set your server name to the `nginx.conf` file and copy it to the `nginx/conf.d` directory (update the port if necessary to match the one in the `docker-compose` file). Use Certbot to generate the SSL certificates.

# Run

1. Build the project:

```bash
docker-compose build
```

2. Start the services:

```bash
docker-compose up -d
```

3. Stop the services:

```bash
docker-compose down
```

# Access the services:

-   Load Balancer: http://localhost:8000
-   Prometheus: http://localhost:9091
-   Grafana: http://localhost:3000 (login with admin/admin)

Once connected to Grafana, you can import the dashboard located at `grafana/provisioning/dashboards/loadbalancer.json`, modify it, or create your own.

# Tests

## Test redirection / simulate load

1. Create 5 dumb servers:

```bash
go run test/server/server.go --num-servers=5
```

2. Add these servers to the database:

-   http://localhost:8080
-   http://localhost:8081
    ...

3. Run the load balancer:

```bash
go run ./src dev
```

# Simulate load

## Using Apache Benchmark

Create a load test with Apache Benchmark:

10000 requests with 10 concurrent requests:

```bash
ab -n 10000 -c 10 http://localhost:8000/
```

## Using custom script

Apache benchmark does not allow to process a group of requests every second. To simulate this, you can use the following script:

```bash
go run test/loadtest.go --url=http://localhost:8000/ -n=10000 -c=10 -d=1
```

Parameters:

-   `url`: URL to test
-   `n`: Number of requests
-   `c`: Number of concurrent requests
-   `d`: Delay between requests (in seconds) [Default = 1]
