package main

import (
	"fmt"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

const defaultPollFrequency = time.Millisecond // 1/ms

func main() {
	err := rpio.Open()
	if err != nil {
		panic(err)
	}
	defer rpio.Close()

	pin := rpio.Pin(14)
	pin.Input() // set pin as input
	//pin.PullDown() // default to signal of zero when no flow
	pin.PullUp()
	pin.Detect(rpio.RiseEdge)

	fmt.Println("read:", pin.Read())

	quit := make(chan struct{})
	var count int
	callback := func() {
		count += 1
	}

	poll(pin, defaultPollFrequency, quit, callback)

	fmt.Println("signals:", count)
	pin.Detect(rpio.NoEdge)
}

func poll(pin rpio.Pin, freq time.Duration, quit chan struct{}, callback func()) {
	lastTime := time.Now()
	select {
	case <-quit:
		return
	default:
		if pin.EdgeDetected() {
			go callback()
		}
		time.Sleep(freq - time.Now().Sub(lastTime))
	}
}
