package server

import (
	"encoding/json"
	"log"
	"os"
)

// Interval represents the time interval configuration
type Interval struct {
    Unit  string `json:"unit"`
    Value int    `json:"value"`
}


// Request represents the HTTP request configuration
type Request struct {
    Method string      `json:"method"`
    Body   struct{} `json:"body"`
}

// HealthCheck represents the main configuration structure
type HealthCheck struct {
    Interval Interval `json:"interval"`
    Request  Request  `json:"request"`
}

// Config is the root configuration structure
type HealthConfig struct {
    HealthCheck HealthCheck `json:"healthCheck"`
}

// Validate checks if the configuration is valid
func (c *HealthConfig) Validate() {
    // Validate interval unit
    validUnits := map[string]bool{
        "hour":   true,
        "minute": true,
        "second": true,
    }
    if !validUnits[c.HealthCheck.Interval.Unit] {
        log.Fatalf("invalid interval unit: %s. Must be one of: hour, minute, second", 
            c.HealthCheck.Interval.Unit)
    }

    // Validate interval value
    if c.HealthCheck.Interval.Value <= 0 {
        log.Fatalf("interval value must be positive, got: %d", 
            c.HealthCheck.Interval.Value)
    }

	if c.HealthCheck.Interval.Unit == "hour" &&  c.HealthCheck.Interval.Value > 24 {
		log.Fatalf("interval value cannot be greater than 24 for hour unit")
	}

    // Validate HTTP method
    validMethods := map[string]bool{
        "POST": true,
        "GET":  true,
    }
    if !validMethods[c.HealthCheck.Request.Method] {
        log.Fatalf("invalid HTTP method: %s. Must be either POST or GET", 
            c.HealthCheck.Request.Method)
    }
}

func loadHealthConfig() HealthConfig {
	// Read the configuration file
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Parse the configuration
	var config HealthConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Error parsing config: %v", err)
	}

	// Validate the configuration
	config.Validate()

	return config
}

