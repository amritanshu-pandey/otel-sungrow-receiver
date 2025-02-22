package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/goburrow/modbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Modbus TCP settings
const (
	inverterAddress = "10.0.20.110:502" // Change this to your inverter's IP and port
	slaveID        = 1                    // Usually 1, but verify with your inverter
	readInterval   = 10 * time.Second      // How often to poll the inverter
)

// Prometheus metrics
var (
	dailyPower = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sungrow_daily_power_yield",
		Help: "Current power output of the Sungrow inverter in watts",
	})
	totalPower = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sungrow_total_power_yield",
		Help: "DC voltage of the Sungrow inverter in volts",
	})
)

// Initialize the Modbus client
func newModbusClient() modbus.Client {
	handler := modbus.NewTCPClientHandler(inverterAddress)
	handler.Timeout = 5 * time.Second
	handler.SlaveId = byte(slaveID)

	err := handler.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to Modbus device: %v", err)
	}

	return modbus.NewClient(handler)
}

// Read data from the inverter
func pollInverter(client modbus.Client) {
	for {
		// Replace these with actual Modbus registers from your Sungrow inverter documentation
		// dailyPowerYield := uint16(5003)
		// totalPowerYield := uint16(5004)

		results, err := client.ReadInputRegisters(5002, 1)
		if err != nil {
			fmt.Print(err)
		}

		if err == nil {
			value := float64(uint32(results[0])<<8 | uint32(results[1])) / 10
			dailyPower.Set(value)
			fmt.Printf("Daily Power Yield: %.2f kWh\n", value)
		}

		results, err = client.ReadInputRegisters(5003, 2)
		if err != nil {
			fmt.Print(err)
		}
		if err == nil {
			value := float64(uint32(results[0])<<8 | uint32(results[1]))
			totalPower.Set(value)
			fmt.Printf("Total Power Yield: %v kWh\n", value)
		}

		time.Sleep(readInterval)
	}
}

func main() {
	// Register Prometheus metrics
	prometheus.MustRegister(dailyPower, totalPower)

	// Start Modbus polling in a separate goroutine
	client := newModbusClient()
	go pollInverter(client)

	// Start HTTP server for Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Serving metrics on :9100/metrics")
	log.Fatal(http.ListenAndServe(":9100", nil))
}
