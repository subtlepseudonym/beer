package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/warthog618/gpio"
)

const (
	defaultDeltaThreshold     = time.Second // used to separate pour events
	defaultPourEventThreshold = 10          // number of flow events to exceed to constitute a pour
)

type FlowMeter struct {
	Model        string  `json:"model"`
	FlowConstant float64 `json:"flow_constant"`
}

// These values are here for reference, but are not actually used
// Actual FlowConstant values are loaded from file
var (
	// digiten fl-s401a
	FlowMeterFLS401A = FlowMeter{
		Model:        "fl-s401a",
		FlowConstant: 98,
	}
	// gredia gr-r401
	FlowMeterGRR401 = FlowMeter{
		Model:        "gr-r401",
		FlowConstant: 98,
	}
	// gredia gr-301
	FlowMeterGR301 = FlowMeter{
		Model:        "gr-301",
		FlowConstant: 21,
	}
	// uxcell a18041200ux0151
	FlowMeterUX0151 = FlowMeter{
		Model:        "ux0151",
		FlowConstant: 76,
	}
)

type Pour struct {
	prune  *time.Timer `json:"-"`
	events int         `json:"-"`
	keg    string      `json:"keg"`

	StartTime time.Time     `json:"time"`
	Duration  time.Duration `json:"duration"`
	Volume    float64       `json:"volume"`
}

func (p Pour) MarshalJSON() ([]byte, error) {
	pour := struct {
		Time     string  `json:"time"`
		Keg      string  `json:"keg"`
		Duration float64 `json:"duration"`
		Volume   float64 `json:"volume"`
	}{
		Time:     p.StartTime.Format(time.RFC3339),
		Keg:      p.keg,
		Duration: p.Duration.Seconds(),
		Volume:   p.Volume,
	}

	return json.Marshal(pour)
}

type Flow struct {
	keg       *Keg
	sensor    *FlowMeter
	pinNumber int
	pin       *gpio.Pin

	deltaThreshold time.Duration
	flowPerEvent   float64 // 1 / (flowConstant * 60 seconds)

	mu         sync.Mutex
	signalChan chan int64
	stop       chan struct{}

	latestEvent int64 // microseconds
	eventTotal  int   // scalar
	firstRun    sync.Once

	Pours    []Pour
	Contents string
}

// NewFlow initializes a Flow struct given a flow constant (defined by the flow meter)
// and a starting volume in liters
func NewFlow(flowMeter *FlowMeter, keg *Keg, contents string) *Flow {
	meter := &Flow{
		keg:            keg,
		sensor:         flowMeter,
		deltaThreshold: defaultDeltaThreshold,
		flowPerEvent:   1.0 / (flowMeter.FlowConstant * 60.0),
		signalChan:     make(chan int64, 1000),
		Contents:       contents,
	}

	return meter
}

// Attach allocates a memory range for gpio operations and opens the specified pin for
// input and begins watching it for events
func (f *Flow) Attach(pin uint8) error {
	f.pinNumber = int(pin)
	f.pin = gpio.NewPin(pin)
	f.pin.Input()
	f.pin.PullUp()

	f.pin.Unwatch()
	err := f.pin.Watch(gpio.EdgeRising, func(p *gpio.Pin) {
		now := time.Now()
		f.signalChan <- now.UnixMicro()
	})
	if err != nil {
		return fmt.Errorf("watch pin %d failed: %w", pin, err)
	}

	return nil
}

// Detach releases the memory range held by the gpio package and stops watching
// the signal pin specified by a previous call to attach()
func (f *Flow) Detach() error {
	f.pin.Unwatch()
	return nil
}

// Start reads from the signal channel, updating metrics as each signal is processed
func (f *Flow) Start() {
	if f.stop != nil {
		return
	}

	f.stop = make(chan struct{}, 1)
	go func() {
		for {
			select {
			case event := <-f.signalChan:
				f.update(event)
			case <-f.stop:
				return
			}
		}
	}()
}

// Stop stops monitoring keg liquid flow
func (f *Flow) Stop() {
	if f.stop == nil {
		return
	}
	close(f.stop)
	f.pin.Unwatch()
}

func (f *Flow) Lock() {
	f.mu.Lock()
}

func (f *Flow) Unlock() {
	f.mu.Unlock()
}

// Refill resets state with a new contents value
func (f *Flow) Refill(contents string) {
	f.mu.Lock()
	f.Contents = contents
	f.eventTotal = 0
	f.mu.Unlock()
}

// TotalFlow is a convenience method for determining the total volume of flow, in
// liters, that have been measured
func (f *Flow) TotalFlow() float64 {
	return f.flowPerEvent * float64(f.eventTotal)
}

// RemainingVolume is a convenience method for reporting the total volume remaining
// in the keg
func (f *Flow) RemainingVolume() float64 {
	return f.keg.Volume - f.TotalFlow()
}

// Keg returns a struct containing keg name and volume
func (f *Flow) Keg() *Keg {
	return f.keg
}

// Sensor returns a struct containing flow meter model and flow constant
func (f *Flow) Sensor() *FlowMeter {
	return f.sensor
}

// Pin returns the pin number that the flow meter is attached to
func (f *Flow) Pin() int {
	return f.pinNumber
}

// update calculates the current flow rate and pour amount
//
// Each pulse from the flow meter indicates a specific amount of flow. Flow rate
// can be calculated by counting the number of pulses per unit time.
//
// Flow rate formula provided is printed on the side of the Gredia flow meter:
// F = 21Q; F is number of pulses; Q is liters / minute
func (f *Flow) update(event int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delta := time.Duration(event-f.latestEvent) * time.Microsecond

	// TODO: make atomic / thread-safe
	f.latestEvent = event
	f.eventTotal += 1

	// Only update flow rate if there's an ongoing pour
	if delta > f.deltaThreshold {
		idx := len(f.Pours)
		prune := time.AfterFunc(f.deltaThreshold, func() {
			f.mu.Lock()
			defer f.mu.Unlock()

			if idx >= len(f.Pours) {
				log.Printf("WARN: prune index %d greater than highest pour index\n", idx)
				return
			} else if idx < 0 {
				log.Printf("WARN: negative prune index: %d\n", idx)
				return
			}
			f.eventTotal -= f.Pours[idx].events
			f.Pours = append(f.Pours[:idx], f.Pours[idx+1:]...)

			if len(f.Pours) == 0 {
				f.latestEvent = 0
			} else {
				latestPour := f.Pours[idx-1]
				f.latestEvent = latestPour.StartTime.Add(latestPour.Duration).UnixMicro()
			}
		})

		f.Pours = append(f.Pours, Pour{
			prune:     prune,
			events:    1,
			keg:       fmt.Sprintf("%d_%s", f.pinNumber, f.Contents),
			StartTime: time.UnixMicro(event),
		})
		return
	}

	idx := len(f.Pours) - 1
	f.Pours[idx].events += 1
	f.Pours[idx].Duration += delta
	f.Pours[idx].Volume += f.flowPerEvent

	pour := f.Pours[idx]
	if pour.events < defaultPourEventThreshold {
		pour.prune.Reset(f.deltaThreshold)
	} else if pour.events == defaultPourEventThreshold {
		// once pour threshold is reached, stop prune goroutine
		pour.prune.Stop()

		poured := float64(defaultPourEventThreshold) * f.flowPerEvent
		PourVolume.WithLabelValues(
			strconv.Itoa(f.pinNumber),
			f.keg.Type,
			f.Contents,
		).Add(poured)
	} else {
		PourVolume.WithLabelValues(
			strconv.Itoa(f.pinNumber),
			f.keg.Type,
			f.Contents,
		).Add(f.flowPerEvent)
	}
}
