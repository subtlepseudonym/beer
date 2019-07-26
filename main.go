package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/warthog618/gpio"
)

const (
	defaultDeltaThreshold = time.Second
	defaultFlowConstant   = 23 // 1 / liters
)

type FlowMeter struct {
	deltaThreshold time.Duration
	flowConstant   float64 // scalar
	pin            *gpio.Pin

	latestEvent    time.Time
	totalFrequency float64 // 1 / seconds
	totalFlowRate  float64 // liters / seconds
	// TODO: convert these to math/big.Float

	TotalEvents     int // scalar
	TotalPourEvents int // scalar
	TotalPourTime   time.Duration
	TotalPour       float64 // liters
	RemainingVolume float64 // liters
}

func NewFlowMeter(filePath string, flowConstant float64) *FlowMeter {
	// TODO: read initial state from json file

	flowMeter := &FlowMeter{
		deltaThreshold:  defaultDeltaThreshold,
		flowConstant:    defaultFlowConstant,
		TotalPourEvents: -1,
	}

	if flowConstant > 0 {
		flowMeter.flowConstant = flowConstant
	}

	return flowMeter
}

// Attach allocates a memory range for gpio operations and opens the specified pin for
// input and begins watching it for events
func (f *FlowMeter) Attach(pin uint8) error {
	f.pin = gpio.NewPin(pin)
	f.pin.Input()
	f.pin.PullUp()

	f.pin.Unwatch()
	err := f.pin.Watch(gpio.EdgeRising, func(p *gpio.Pin) {
		f.update(time.Now())
	})
	if err != nil {
		return errors.Wrapf(err, "watch pin %d failed", pin)
	}

	return nil
}

// Detach releases the memory range held by the gpio package and stops watching
// the signal pin specified by a previous call to Attach()
func (f *FlowMeter) Detach() error {
	f.pin.Unwatch()
	return nil
}

func (f *FlowMeter) update(now time.Time) {
	delta := now.Sub(f.latestEvent)

	if delta > f.deltaThreshold {
		f.TotalPourEvents += 1
		f.latestEvent = now
		return
	}

	f.TotalEvents += 1

	frequency := 1.0 / delta.Seconds()
	f.totalFrequency += frequency // used to calculate average frequency
	f.TotalPourTime += delta

	pour := 1.0 / (time.Minute.Seconds() * f.flowConstant)
	f.TotalPour += pour
	f.RemainingVolume -= pour

	flowRate := frequency * pour
	f.totalFlowRate += flowRate // used to calculate average flow rate

	f.latestEvent = now
}

func main() {
	err := gpio.Open()
	if err != nil {
		panic(err)
	}
	defer gpio.Close()

	meter := NewFlowMeter("file", defaultFlowConstant)
	err = meter.Attach(14)
	if err != nil {
		panic(err)
	}
	defer meter.Detach()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	defer signal.Stop(quit)

	fmt.Println(`{"msg":"listening..."}`)
	for {
		select {
		case <-time.After(time.Second):
			b, _ := json.Marshal(meter)
			fmt.Printf("%s\n", b)
		case <-time.After(time.Minute):
		case <-quit:
			return
		}
	}
}
