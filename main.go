package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/warthog618/gpio"
)

const (
	defaultDeltaThreshold = time.Second // used to separate pour events
	defaultGPIOPin        = 14          // pins are zero-index, so this is pin 15
)

const (
	FlowConstantGR301 = 21 // flow constant of the Gredia GR-301, in (1 / liters)

	VolumeCorny   = 18.93 // cornelius
	VolumeSixtel  = 19.55 // sixth-barrel
	VolumeQuarter = 29.34 // pony
	VolumeHalf    = 58.67 // full size
)

type Flow struct {
	deltaThreshold time.Duration
	startingVolume float64 // scalar
	flowConstant   float64 // scalar
	flowPerEvent   float64 // 1 / (flowConstant * 60 seconds)

	pin        *gpio.Pin
	signalChan chan int64

	latestEvent int64 // microseconds
	eventTotal  int   // scalar

	Pours []Pour
}

type Pour struct {
	StartTime time.Time
	Duration  time.Duration
	Events    int64
}

// NewFlow initializes a Flow struct given a flow constant (defined by the flow meter)
// and a starting volume in liters
func NewFlow(flowConstant, startingVolume float64) *Flow {
	meter := &Flow{
		deltaThreshold: defaultDeltaThreshold,
		startingVolume: startingVolume,
		flowConstant:   flowConstant,
		flowPerEvent:   1.0 / (flowConstant * 60.0),
		signalChan:     make(chan int64, 1000),
	}

	return meter
}

// Attach allocates a memory range for gpio operations and opens the specified pin for
// input and begins watching it for events
func (f *Flow) Attach(pin uint8) error {
	f.pin = gpio.NewPin(pin)
	f.pin.Input()
	f.pin.PullUp()

	f.pin.Unwatch()
	err := f.pin.Watch(gpio.EdgeRising, func(p *gpio.Pin) {
		now := time.Now()
		f.signalChan <- now.UnixMicro()
	})
	if err != nil {
		return errors.Wrapf(err, "watch pin %d failed", pin)
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
func (f *Flow) Start() chan struct{} {
	quit := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case event := <-f.signalChan:
				f.update(event)
			case <-quit:
				return
			}
		}
	}()
	return quit
}

// TotalFlow is a convenience method for determining the total volume of flow, in
// liters, that have been measured
func (f *Flow) TotalFlow() float64 {
	return f.flowPerEvent * float64(f.eventTotal)
}

// RemainingVolume is a convenience method for reporting the total volume remaining
// in the keg
func (f *Flow) RemainingVolume() float64 {
	return f.startingVolume - f.TotalFlow()
}

// update calculates the current flow rate and pour amount
//
// Each pulse from the flow meter indicates a specific amount of flow. Flow rate
// can be calculated by counting the number of pulses per unit time.
//
// Flow rate formula provided is printed on the side of the Gredia flow meter:
// F = 21Q; F is number of pulses; Q is liters / minute
func (f *Flow) update(event int64) {
	delta := time.Duration(event-f.latestEvent) * time.Microsecond

	// TODO: make atomic / thread-safe
	f.latestEvent = event
	f.eventTotal += 1

	// Only update flow rate if there's an ongoing pour
	if delta > f.deltaThreshold {
		pour := Pour{
			StartTime: time.UnixMicro(event),
			Events:    1,
		}
		f.Pours = append(f.Pours, pour)
	} else {
		pour := f.Pours[len(f.Pours)-1]
		pour.Duration += delta
		pour.Events += 1
	}
}

func main() {
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()

	meter := NewFlow(FlowConstantGR301, VolumeSixtel)
	err = meter.Attach(14)
	if err != nil {
		panic(err)
	}
	defer meter.Detach()

	stop := meter.Start()
	defer close(stop)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	defer signal.Stop(quit)

	for {
		select {
		case <-time.After(time.Second):
			fmt.Printf("Events: % 5d; Pours: % 2d; Volume: % 2.4f\n", meter.eventTotal, len(meter.Pours), meter.TotalFlow())
		case <-time.After(time.Minute):
		case <-quit:
			return
		}
	}
}
