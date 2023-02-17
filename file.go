package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"

	dhtlib "github.com/d2r2/go-dht"
)

// State maintains the current state of kegs and DHT sensors in the fridge
// and is used for both saving state to file and writing state to a REST
// endpoint
type State struct {
	mu   sync.Mutex `json:"-"`
	kegs []*Flow    `json:"-"`
	dhts []*DHT     `json:"-"`

	Kegs []KegState `json:"kegs"`
	DHTs []DHTState `json:"dhts"`
}

// Update ensures that the exported state fields represent the state's
// internal representation
func (s *State) update() {
	kegStates := make([]KegState, len(s.kegs))
	for i, keg := range s.kegs {
		keg.Lock()
		pours := make([]float64, len(keg.Pours))
		for i := range keg.Pours {
			pours[i] = float64(keg.Pours[i].Events) * keg.flowPerEvent
		}
		kegState := KegState{
			Keg:      keg.keg,
			Contents: keg.Contents,
			Sensor:   keg.SensorModel(),
			Pin:      keg.Pin(),
			Poured:   keg.TotalFlow(),
			Pours:    pours,
		}
		keg.Unlock()
		kegStates[i] = kegState
	}

	dhtStates := make([]DHTState, len(s.dhts))
	for i, dht := range s.dhts {
		dht.Lock()
		dhtState := DHTState{
			Model:       dht.Model(),
			Pin:         dht.pin,
			Temperature: dht.Temperature,
			Humidity:    dht.Humidity,
		}
		dht.Unlock()
		dhtStates[i] = dhtState
	}

	s.Kegs = kegStates
	s.DHTs = dhtStates
}

type KegState struct {
	Keg      *Keg      `json:"keg"`
	Contents string    `json:"contents"`
	Sensor   string    `json:"sensor"`
	Pin      int       `json:"pin"`
	Poured   float64   `json:"poured"`
	Pours    []float64 `json:"pours,omitempty"`
}

type DHTState struct {
	Model       string  `json:"model"`
	Pin         int     `json:"pin"`
	Temperature float32 `json:"temperature,omitempty"`
	Humidity    float32 `json:"humidity,omitempty"`
}

var (
	// FIXME: using state.json as a key map is lazy
	kegSizes map[string]*Keg = map[string]*Keg{
		"corny":       &KegCorny,
		"sixtel":      &KegSixtel,
		"quarter":     &KegQuarter,
		"half-barrel": &KegHalf,
	}
	flowSensors map[string]*FlowMeter = map[string]*FlowMeter{
		"fl-s401a": &FlowMeterFLS401A,
		"gr-301":   &FlowMeterGR301,
		"ux0151":   &FlowMeterUX0151,
	}
	dhtModels map[string]dhtlib.SensorType = map[string]dhtlib.SensorType{
		"dht22": dhtlib.DHT22,
	}
)

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

	for _, keg := range state.Kegs {
		kegSize, ok := kegSizes[keg.Keg.Type]
		if !ok {
			return nil, fmt.Errorf("invalid keg type %q", keg.Keg.Type)
		}
		flowSensor, ok := flowSensors[keg.Sensor]
		if !ok {
			return nil, fmt.Errorf("invalid flow sensor type %q", keg.Sensor)
		}

		flow := NewFlow(flowSensor, kegSize, keg.Contents)
		flow.eventTotal = int(math.Ceil(keg.Poured / flow.flowPerEvent))
		err = flow.Attach(uint8(keg.Pin % math.MaxUint8))
		if err != nil {
			return nil, fmt.Errorf("attach flow on pin %d: %s", keg.Pin, err)
		}
		state.kegs = append(state.kegs, flow)
	}

	for _, dht := range state.DHTs {
		dhtModel, ok := dhtModels[dht.Model]
		if !ok {
			return nil, fmt.Errorf("invalid dht model %q", dht.Model)
		}

		dhtSensor := NewDHT(dhtModel, defaultDHTReadInterval)
		err = dhtSensor.Attach(dht.Pin)
		if err != nil {
			return nil, fmt.Errorf("attach dht on pin %d: %s", dht.Pin, err)
		}
		state.dhts = append(state.dhts, dhtSensor)
	}

	return &state, nil
}

func SaveStateToFile(filename string, state *State) error {
	state.mu.Lock()
	state.update()

	f, err := os.OpenFile(filename, os.O_RDWR, 0755)
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
