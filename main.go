package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

// Config structure for YAML
type Config struct {
	Modbus struct {
		IP           string `yaml:"ip"`
		Port         int    `yaml:"port"`
		SlaveID      int    `yaml:"slave_id"`
		ReadInterval int    `yaml:"read_interval"`
	} `yaml:"modbus"`
	Metrics []struct {
		Name     string  `yaml:"name"`
		Register uint16  `yaml:"register"`
		Type     string  `yaml:"type"` // "U16", "U32", "S16", "S32"
		Help     string  `yaml:"help"`
		Scale    float64 `yaml:"scale"`
		Round    bool    `yaml:"round"`
		Unit     string  `yaml:"unit"`
	} `yaml:"metrics"`
	SolarFeedInTarrif float64 `yaml:"solar_feed_in_tarrif"`
	GridEnergyTarrif  float64 `yaml:"grid_energy_tarrif"`
}

// LoadConfig reads the YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

// Read Modbus data and update Prometheus metrics
func pollInverter(client modbus.Client, config *Config, metricsMap map[string]prometheus.Gauge, solarFIT prometheus.Gauge, gridTarrif prometheus.Gauge) {
	for {
		for _, metric := range config.Metrics {
			var results []byte
			var err error

			// Read 1 register for 16-bit, 2 registers for 32-bit
			switch metric.Type {
			case "U16", "S16":
				results, err = client.ReadInputRegisters(metric.Register-1, 1)
			case "U32", "S32":
				results, err = client.ReadInputRegisters(metric.Register-1, 2)
			default:
				log.Printf("Unsupported type for metric %s: %s", metric.Name, metric.Type)
				continue
			}

			if err != nil {
				log.Printf("Error reading %s (register %d): %v", metric.Name, metric.Register, err)
				continue
			}

			// Default scale to 1 if not set
			scale := metric.Scale
			if scale == 0 {
				scale = 1
			}

			// Convert the register data to the correct type
			var value float64
			switch metric.Type {
			case "U16":
				value = float64(uint32(results[0])<<8 | uint32(results[1]))
			case "S16":
				rawValue := int16(results[0])<<8 | int16(results[1])
				value = float64(rawValue)
			case "U32":
				value = float64(uint32(results[0])<<8 | uint32(results[1]))
			case "S32":
				rawValue := int32(results[0])<<24 | int32(results[1])<<16 | int32(results[2])<<8 | int32(results[3])
				value = float64(rawValue)
			}

			// Apply scaling
			value *= scale

			// Apply rounding if required
			if metric.Round {
				value = math.Round(value)
			}

			// Update Prometheus metric
			metricsMap[metric.Name].Set(value)
			if metric.Unit != "" {
				fmt.Printf("%s: %.2f %s\n", metric.Name, value, metric.Unit)
			} else {
				fmt.Printf("%s: %.2f\n", metric.Name, value)
			}
		}

		solarFIT.Set(config.SolarFeedInTarrif)
		gridTarrif.Set(config.GridEnergyTarrif)

		time.Sleep(time.Duration(config.Modbus.ReadInterval) * time.Second)
	}
}

func main() {
	// Load configuration
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create Modbus client
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", config.Modbus.IP, config.Modbus.Port))
	handler.Timeout = 5 * time.Second
	handler.SlaveId = byte(config.Modbus.SlaveID)
	err = handler.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to Modbus device: %v", err)
	}
	defer handler.Close()
	client := modbus.NewClient(handler)

	// Initialize Prometheus metrics dynamically
	metricsMap := make(map[string]prometheus.Gauge)
	for _, metric := range config.Metrics {
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: metric.Name,
			Help: metric.Help,
		})
		metricsMap[metric.Name] = gauge
		prometheus.MustRegister(gauge)
	}
	solarFIT := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solar_feed_in_tarrif",
		Help: "Solar feed in tarrif per kWh",
	})
	prometheus.MustRegister(solarFIT)

	gridTarrif := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "grid_energy_tarrif",
		Help: "Grid energy tarrif per kWh",
	})
	prometheus.MustRegister(gridTarrif)

	// Start polling Modbus in a goroutine
	go pollInverter(client, config, metricsMap, solarFIT, gridTarrif)

	// Start HTTP server for Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Serving metrics on 0.0.0.0:9100/metrics")
	log.Fatal(http.ListenAndServe("0.0.0.0:9100", nil))
}
