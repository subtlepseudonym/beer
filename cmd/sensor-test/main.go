package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	keg "github.com/subtlepseudonym/kegerator"
	"github.com/subtlepseudonym/kegerator/prometheus"

	godht "github.com/d2r2/go-dht"
)

const (
	defaultPin            = -1
	defaultFlowMeterModel = "fl-s401a"
	defaultFlowConstant   = 98
	defaultDHTModel       = "dht22"
)

var (
	Version string = "0.0.1-unknown"
)

func main() {
	vFlag := flag.Bool("version", false, "Display version information")
	flowPin := flag.Int("flow", defaultPin, "Flow meter pin to test. If this flag is provided with --dht, only the flow meter will be tested")
	flowModel := flag.String("flow-model", defaultFlowMeterModel, "Flow meter model name")
	flowConstant := flag.Float64("flow-constant", defaultFlowConstant, "Flow meter flow constant")
	dhtPin := flag.Int("dht", defaultPin, "DHT pin to test. If this flag is provided with --flow, only the flow meter will be tested")
	dhtModel := flag.String("dht-model", defaultDHTModel, "DHT model name")
	flag.Parse()

	if *vFlag {
		fmt.Println("sensor-test", Version)
		return
	}

	prometheus.BuildMetrics()

	// Exit gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	if *flowPin > -1 {
		sensor := &keg.FlowMeter{
			Model:        *flowModel,
			FlowConstant: *flowConstant,
		}
		flow := keg.NewFlow(sensor, &keg.KegHalf, "test")

		err := flow.Attach(uint8(*flowPin))
		if err != nil {
			fmt.Println("flow attach:", err)
			return
		}
		flow.Start(flow.Count)

		for {
			select {
			case <-time.After(5 * time.Second):
				flow.Lock()
				fmt.Printf("flow: %.4f\n", flow.TotalFlow())
				flow.Unlock()
			case <-quit:
				flow.Stop()
			}
		}
		return
	}

	if *dhtPin > -1 {
		model, err := keg.GetDHTModel(*dhtModel)
		if err != nil {
			fmt.Println("dht:", err)
			return
		}

		dht := keg.NewDHT(model, 10*time.Second)
		err = dht.Attach(*dhtPin)
		if err != nil {
			fmt.Println("dht attach:", err)
			return
		}
		dht.Start(func(ctx context.Context) {
			temp, humid, retries, err := godht.ReadDHTxxWithContextAndRetry(ctx, model, *dhtPin, false, 4)
			if err != nil {
				fmt.Println("ERR:", err)
				return
			}

			fmt.Printf("-----\ntemperature: %.2f\nhumidity: %.2f\nretries: %d\n", temp, humid, retries)
		})

		<-quit
		dht.Stop()
		return
	}
}
