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
		Type     string  `yaml:"type"` // "U16" or "U32"
		Help     string  `yaml:"help"`
		Scale    float64 `yaml:"scale"`
		Round    bool    `yaml:"round"`
		Unit     string  `yaml:"unit"`
	} `yaml:"metrics"`
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
func pollInverter(client modbus.Client, config *Config, metricsMap map[string]prometheus.Gauge) {
	for {
		for _, metric := range config.Metrics {
			var results []byte
			var err error

			// Read 1 register for U16, 2 registers for U32
			if metric.Type == "U16" {
				results, err = client.ReadInputRegisters(metric.Register-1, 1)
			} else if metric.Type == "U32" {
				results, err = client.ReadInputRegisters(metric.Register-1, 2)
			} else {
				log.Printf("Unsupported type for metric %s: %s", metric.Name, metric.Type)
				continue
			}

			if err != nil {
				log.Printf("Error reading %s (register %d): %v", metric.Name, metric.Register, err)
				continue
			}

			// Parse the results
			scale := metric.Scale
			if metric.Scale == 0 {
				scale = 1
			}
			value := float64(uint32(results[0])<<8|uint32(results[1])) * scale
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

	// Start polling Modbus in a goroutine
	go pollInverter(client, config, metricsMap)

	// Start HTTP server for Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Serving metrics on :9100/metrics")
	log.Fatal(http.ListenAndServe(":9100", nil))
}
