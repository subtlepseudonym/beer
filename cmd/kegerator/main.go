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

	keg "github.com/subtlepseudonym/kegerator"
	"github.com/subtlepseudonym/kegerator/prometheus"

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
	registry := prometheus.BuildMetrics()
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()

	keg.GlobalState, err = keg.LoadStateFromFile(stateFile)
	if err != nil {
		log.Println("ERR:", err)
		return
	}

	for _, keg := range keg.GlobalState.Kegs {
		keg.Start(keg.Update)
	}
	for _, dht := range keg.GlobalState.DHTs {
		dht.Start(dht.Update)
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
				err = keg.SaveStateToFile(stateFile, keg.GlobalState)
				if err != nil {
					log.Println("ERR: save state file:", err)
				}
			case <-reload:
				// stop existing state
				saveTicker.Stop()
				for _, keg := range keg.GlobalState.Kegs {
					keg.Stop()
				}
				for _, dht := range keg.GlobalState.DHTs {
					dht.Stop()
				}

				// load and start new state
				s, err := keg.LoadStateFromFile(stateFile)
				if err != nil {
					log.Println("ERR:", err)
					continue
				}
				for _, keg := range s.Kegs {
					keg.Start(keg.Update)
				}
				for _, dht := range s.DHTs {
					dht.Start(dht.Update)
				}

				// swap to new state
				oldState := keg.GlobalState
				oldState.Lock()
				// remove old keg data
				prometheus.PourVolume.Reset()
				prometheus.RemainingVolume.Reset()
				keg.GlobalState = s
				oldState.Unlock()
				if !noAutosave {
					saveTicker.Reset(defaultSaveInterval)
				}
			case <-interrupt:
				// stop running kegs and dhts on exit
				for _, keg := range keg.GlobalState.Kegs {
					keg.Stop()
				}
				for _, dht := range keg.GlobalState.DHTs {
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
	mux.Handle("/metrics", keg.MetricsHandler(promHandler))
	mux.HandleFunc("/calibrate", keg.CalibrateHandler)
	mux.HandleFunc("/refill", keg.RefillHandler)
	mux.HandleFunc("/pours", keg.PourHandler)
	mux.HandleFunc("/state", keg.StateHandler)
	mux.HandleFunc("/ok", keg.OKHandler)

	srv := &http.Server{
		Addr:    defaultAddr,
		Handler: mux,
	}
	log.Println("listening on", srv.Addr)
	go srv.ListenAndServe()
	<-stop
	srv.Shutdown(context.Background())
}
