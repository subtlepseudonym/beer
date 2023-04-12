package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "kegerator"
)

var (
	PourVolume          *prometheus.CounterVec
	HTTPRequestDuration *prometheus.CounterVec
	DHTRetries          *prometheus.CounterVec
	RemainingVolume     *prometheus.GaugeVec
	DHTTemperature      *prometheus.GaugeVec
	DHTHumidity         *prometheus.GaugeVec
)

func BuildMetrics() *prometheus.Registry {
	registry := prometheus.NewRegistry()

	PourVolume = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "pour_volume_liters",
			Help:      "Volume of liquid poured from a given keg",
		},
		[]string{"pin", "type", "contents"},
	)

	HTTPRequestDuration = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "How long this exporter takes to respond when scraped by prometheus",
		},
		[]string{"handler"},
	)

	DHTRetries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dht_retries_total",
			Help:      "Number of sensor reading retries with sensor label",
		},
		[]string{"pin", "sensor"},
	)

	RemainingVolume = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "remaining_volume_liters",
			Help:      "Volume of liquid remaining in a given keg",
		},
		[]string{"pin", "type", "contents"},
	)

	DHTTemperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "temperature_celsius",
			Help:      "Temperature of the fridge with sensor label",
		},
		[]string{"pin", "sensor"},
	)

	DHTHumidity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "humidity_ratio",
			Help:      "Humidity of the fridge with sensor label",
		},
		[]string{"pin", "sensor"},
	)

	metrics := []prometheus.Collector{
		PourVolume,
		RemainingVolume,
		HTTPRequestDuration,
		DHTTemperature,
		DHTHumidity,
		DHTRetries,
	}

	for _, metric := range metrics {
		registry.MustRegister(metric)
	}

	return registry
}
