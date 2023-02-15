package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func StateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	state.mu.Lock()
	state.update()
	err := json.NewEncoder(w).Encode(state)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("marshal state: %w", err)
		return
	}
	state.mu.Unlock()
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func MetricsHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		handler.ServeHTTP(w, r)
		HTTPRequestDuration.WithLabelValues("/metrics").Add(time.Since(now).Seconds())
	})
}
