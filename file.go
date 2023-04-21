package kegerator

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
)

// GlobalState holds all the keg and sensor state
// This allows the main package as well as http endpoints to modify
// sensor state
//
// FIXME: really ought to unexport this and write more methods to updating state
var GlobalState *State

// State maintains the current state of kegs and DHT sensors in the fridge
// and is used for both saving state to file and writing state to a REST
// endpoint
type State struct {
	mu   sync.Mutex `json:"-"`
	Kegs []*Flow    `json:"-"`
	DHTs []*DHT     `json:"-"`

	KegOut []kegOutput `json:"kegs"`
	DHTOut []dhtOutput `json:"dhts"`
}

func (s *State) Lock() {
	s.mu.Lock()
}

func (s *State) Unlock() {
	s.mu.Unlock()
}

// Update ensures that the exported state fields represent the state's
// internal representation
func (s *State) update() {
	kegOutputs := make([]kegOutput, len(s.Kegs))
	for i, keg := range s.Kegs {
		keg.Lock()
		out := kegOutput{
			Keg:      keg.keg,
			Contents: keg.Contents,
			Sensor:   keg.Sensor(),
			Pin:      keg.Pin(),
			Poured:   keg.TotalFlow(),
		}
		keg.Unlock()
		kegOutputs[i] = out
	}

	dhtOutputs := make([]dhtOutput, len(s.DHTs))
	for i, dht := range s.DHTs {
		dht.Lock()
		out := dhtOutput{
			Model:       dht.Model(),
			Pin:         dht.pin,
			Temperature: dht.Temperature,
			Humidity:    dht.Humidity,
		}
		dht.Unlock()
		dhtOutputs[i] = out
	}

	s.KegOut = kegOutputs
	s.DHTOut = dhtOutputs
}

type kegOutput struct {
	Keg      *Keg       `json:"keg"`
	Sensor   *FlowMeter `json:"sensor"`
	Contents string     `json:"contents"`
	Pin      int        `json:"pin"`
	Poured   float64    `json:"poured"`
}

type dhtOutput struct {
	Model       string  `json:"model"`
	Pin         int     `json:"pin"`
	Temperature float32 `json:"temperature,omitempty"`
	Humidity    float32 `json:"humidity,omitempty"`
}

func LoadStateFromFile(filename string) (*State, error) {
	var state State
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open state file: %w", err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&state)
	if err != nil {
		return nil, fmt.Errorf("decode state file: %w", err)
	}

	for _, keg := range state.KegOut {
		flow := NewFlow(keg.Sensor, keg.Keg, keg.Contents)
		flow.eventTotal = int(math.Ceil(keg.Poured / flow.flowPerEvent))
		err = flow.Attach(uint8(keg.Pin % math.MaxUint8))
		if err != nil {
			return nil, fmt.Errorf("attach flow on pin %d: %s", keg.Pin, err)
		}
		state.Kegs = append(state.Kegs, flow)
	}

	for _, dht := range state.DHTOut {
		dhtModel, ok := dhtModels[dht.Model]
		if !ok {
			return nil, fmt.Errorf("invalid dht model %q", dht.Model)
		}

		dhtSensor := NewDHT(dhtModel, defaultDHTReadInterval)
		err = dhtSensor.Attach(dht.Pin)
		if err != nil {
			return nil, fmt.Errorf("attach dht on pin %d: %s", dht.Pin, err)
		}
		state.DHTs = append(state.DHTs, dhtSensor)
	}

	return &state, nil
}

func SaveStateToFile(filename string, state *State) error {
	state.mu.Lock()
	state.update()

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("open state file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(state)
	if err != nil {
		return fmt.Errorf("encode state file: %w", err)
	}
	state.mu.Unlock()

	return nil
}
