# Sungrow Modbus TCP Exporter

## Overview
This Go-based exporter reads metrics from a Sungrow inverter using Modbus TCP and exposes them in Prometheus format.

## Features
- Connects to a Sungrow inverter via Modbus TCP
- Reads configured registers at a set interval
- Scales, rounds, and formats values according to the configuration
- Exposes metrics in Prometheus format
- Runs an HTTP server to serve metrics

## Requirements
- Go (1.18+)
- Prometheus
- Sungrow inverter with Modbus TCP enabled

## Installation
1. Clone this repository:
   ```sh
   git clone <repository-url>
   cd modbus-prometheus-exporter
   ```
2. Build the exporter:
   ```sh
   go build -o sungrow_exporter
   ```
3. Ensure your Sungrow inverter has Modbus TCP enabled.

## Configuration
Create a `config.yaml` file with the following structure:

```yaml
modbus:
  ip: "<ip>"
  port: 502
  slave_id: 1
  read_interval: 30  # Polling interval in seconds

grid_energy_tarrif: 0.3 # In your currency
solar_feed_in_tarrif: 0.03 # In your currencyu

metrics:
  - name: "sungrow_arm_software_version"
    register: 4954
    type: "U16"
    help: "ARM software version"
  - name: "sungrow_dsp_software_version"
    register: 4969
    type: "U16"
    help: "DSP software version"
  - name: "sungrow_nominal_active_power"
    register: 5001
    type: "U16"
    help: "Nominal active power"
    unit: "kW"
    scale: 0.1
```

## Running the Exporter
Start the exporter with:
```sh
./sungrow_exporter
```

## Prometheus Metrics Endpoint
The exporter runs an HTTP server on port `9100` by default. Prometheus can scrape metrics from:
```
http://<exporter-host>:9100/metrics
```

## Example Prometheus Configuration
Add the following job to your Prometheus `prometheus.yml`:
```yaml
scrape_configs:
  - job_name: 'sungrow'
    static_configs:
      - targets: ['localhost:9100']
```