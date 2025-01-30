package main

import (
	"net/http"
)

type MuxServer struct {
	*http.ServeMux
}

// Overrides the default HandlerFunc to add a panic recovery mechanism
func (m *MuxServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.ServeMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		handler(w, r)
	})
}

func (m *MuxServer) HandleAuthFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(pattern, auth(handler))
}

func (m *MuxServer) HandleAuthAdminFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(pattern, authAdmin(handler))
}