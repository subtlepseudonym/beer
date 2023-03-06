package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

const defaultPourLimit = 100

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
		log.Printf("marshal state: %s", err)
		return
	}
	state.mu.Unlock()
}

func PourHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	limit := defaultPourLimit
	if r.FormValue("limit") != "" {
		var err error
		limit, err = strconv.Atoi(r.FormValue("limit"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	state.mu.Lock()
	var pours []Pour
	for _, keg := range state.kegs {
		pours = append(pours, keg.Pours...)
	}
	state.mu.Unlock()

	sort.Slice(pours, func(i, j int) bool {
		return pours[i].StartTime.After(pours[j].StartTime)
	})
	if len(pours) > limit {
		pours = pours[:limit]
	}

	err := json.NewEncoder(w).Encode(pours)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("marshal state: %s", err)
		return
	}
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func MetricsHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		state.mu.Lock()
		for _, keg := range state.kegs {
			keg.mu.Lock()
			RemainingVolume.WithLabelValues(
				strconv.Itoa(keg.Pin()),
				keg.Keg().Type,
				keg.Contents,
			).Set(keg.RemainingVolume())
			keg.mu.Unlock()
		}
		state.mu.Unlock()
		handler.ServeHTTP(w, r)
		HTTPRequestDuration.WithLabelValues("/metrics").Add(time.Since(now).Seconds())
	})
}
