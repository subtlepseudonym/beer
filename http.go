package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
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

func RefillHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var err error
	var pin int
	if r.FormValue("pin") != "" {
		pin, err = strconv.Atoi(r.FormValue("pin"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"msg": "bad pin value": "error": %q}`, err)))
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "pin query param required"}`))
		return
	}

	var flow *Flow
	state.mu.Lock()
	for _, keg := range state.kegs {
		if keg.pinNumber == pin {
			flow = keg
			break
		}
	}
	state.mu.Unlock()
	if flow == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "no keg found on pin %d"}`, pin)))
		return
	}

	contents := flow.Contents
	if r.FormValue("contents") != "" {
		contents = r.FormValue("contents")
	} else {
		log.Printf("WARN: refilling %d with existing contents: %s", pin, contents)
	}

	state.mu.Lock()
	flow.Refill(contents)
	state.mu.Unlock()
}

func CalibrateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var err error
	var pin int
	if r.FormValue("pin") != "" {
		pin, err = strconv.Atoi(r.FormValue("pin"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"msg": "bad pin value": "error": %q}`, err)))
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "pin query param required"}`))
		return
	}

	var flow *Flow
	state.mu.Lock()
	for _, keg := range state.kegs {
		if keg.pinNumber == pin {
			flow = keg
			break
		}
	}
	state.mu.Unlock()
	if flow == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error": "no keg found on pin %d"}`, pin)))
		return
	}

	var constant float64
	if r.FormValue("constant") != "" {
		constant, err = strconv.ParseFloat(r.FormValue("constant"), 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"msg": "bad constant value": "error": %q}`, err)))
			return
		}
	} else if r.FormValue("coefficient") != "" {
		coef, err := strconv.ParseFloat(r.FormValue("coefficient"), 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"msg": "bad constant value": "error": %q}`, err)))
			return
		}
		// round to 2 decimal places
		constant = math.Floor(flow.sensor.FlowConstant*coef*100) / 100

		if constant == flow.sensor.FlowConstant {
			log.Printf("WARN: %d flow constant unchanged: %.2f", pin, constant)
			return
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "constant or coefficient query param required"}`))
		return
	}

	state.mu.Lock()
	flow.sensor.FlowConstant = constant
	flow.flowPerEvent = 1.0 / (constant * 60.0)
	state.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
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
