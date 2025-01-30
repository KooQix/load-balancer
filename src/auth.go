package main

import (
	"net/http"
	"net/url"
	"strings"
)

// auth is a middleware that checks if the request has a valid API key

func auth(next http.HandlerFunc)  http.HandlerFunc{
	// API key can be set as query parameter "key" or in the headers "Authorization"
	return func(w http.ResponseWriter, r *http.Request) {

		// Get query parameters
		queryParams, err := url.ParseQuery(r.URL.RawQuery)
	
		if err != nil {
			http.Error(w, "Invalid query parameters", http.StatusBadRequest)
			return
		}
	
		// Check if the API key is in the query parameters
		if apiKey, ok := queryParams["key"]; ok {
			if apiKey[0] != API_KEY {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
	
			// Remove the api key from the query parameters
			queryParams.Del("key")
			r.URL.RawQuery = queryParams.Encode()
		} else {
			// Check if the API key is in the headers
			authKey := strings.Split(r.Header.Get("Authorization"), " ")
			if len(authKey) != 2 || authKey[0] != "Bearer" {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			if authKey[1] != API_KEY {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
		}
	
		// Call the next handler
		next(w, r)
	}

}

func authAdmin(next http.HandlerFunc) http.HandlerFunc {
	// API key can be set in the headers "X-API-KEY"
	return func(w http.ResponseWriter, r *http.Request) {
		
		// Check if the API key is in the headers
		authKey := strings.Split(r.Header.Get("Authorization"), " ")
		if len(authKey) != 2 || authKey[0] != "Bearer" {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		if authKey[1] != ADMIN_API_KEY {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}
	
		// Call the next handler
		next(w, r)
	}
}