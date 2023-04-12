package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/warthog618/gpio"
)

const (
	defaultAddr         = ":9220"
	defaultTimeout      = 5 * time.Second
	defaultSaveInterval = 5 * time.Minute
)

var (
	Version string = "0.0.1-unknown"

	noAutosave bool // prevent automatic saving of state to file
	stateFile  string
	state      *State // storing state as main pkg var so /state can access it
)

func main() {
	vFlag := flag.Bool("version", false, "Display version information")
	flag.BoolVar(&noAutosave, "no-autosave", false, "Do not automatically save state")
	flag.StringVar(&stateFile, "file", "state.json", "File to load initial state from")
	flag.Parse()

	if *vFlag {
		fmt.Println("kegerator", Version)
		return
	}

	// register metrics and prep gpio memory addresses before attaching sensors
	registry := buildMetrics()
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()

	state, err = LoadStateFromFile(stateFile)
	if err != nil {
		log.Println("ERR:", err)
		return
	}

	for _, keg := range state.kegs {
		keg.Start()
	}
	for _, dht := range state.dhts {
		dht.Start()
	}

	// stop any periodic processes on interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	stop := make(chan struct{})

	go func() {
		saveTicker := time.NewTicker(defaultSaveInterval) // save state every 5 minutes
		if noAutosave {
			saveTicker.Stop()
		}
		reload := make(chan os.Signal, 1) // reload state from file on sighup
		signal.Notify(reload, syscall.SIGHUP)

		for {
			select {
			case <-saveTicker.C:
				err = SaveStateToFile(stateFile, state)
				if err != nil {
					log.Println("ERR: save state file:", err)
				}
			case <-reload:
				// stop existing state
				saveTicker.Stop()
				for _, keg := range state.kegs {
					keg.Stop()
				}
				for _, dht := range state.dhts {
					dht.Stop()
				}

				// load and start new state
				s, err := LoadStateFromFile(stateFile)
				if err != nil {
					log.Println("ERR:", err)
					continue
				}
				for _, keg := range s.kegs {
					keg.Start()
				}
				for _, dht := range s.dhts {
					dht.Start()
				}

				// swap to new state
				mu := state.mu
				mu.Lock()
				state = s
				// remove old keg data
				PourVolume.Reset()
				RemainingVolume.Reset()
				mu.Unlock()
				if !noAutosave {
					saveTicker.Reset(defaultSaveInterval)
				}
			case <-interrupt:
				// stop running kegs and dhts on exit
				for _, keg := range state.kegs {
					keg.Stop()
				}
				for _, dht := range state.dhts {
					dht.Stop()
				}
				close(stop)
				return
			}
		}
	}()

	promOpts := promhttp.HandlerOpts{
		Registry: registry,
		Timeout:  defaultTimeout,
	}
	promHandler := promhttp.HandlerFor(registry, promOpts)

	mux := http.NewServeMux()
	mux.Handle("/metrics", MetricsHandler(promHandler))
	mux.HandleFunc("/calibrate", CalibrateHandler)
	mux.HandleFunc("/refill", RefillHandler)
	mux.HandleFunc("/pours", PourHandler)
	mux.HandleFunc("/state", StateHandler)
	mux.HandleFunc("/ok", okHandler)

	srv := &http.Server{
		Addr:    defaultAddr,
		Handler: mux,
	}
	log.Println("listening on", srv.Addr)
	go srv.ListenAndServe()
	<-stop
	srv.Shutdown(context.Background())
}
