# RaspberryWaterSystem

![CI - make test](https://github.com/bruli/raspberryWaterSystem/actions/workflows/test.yml/badge.svg)
![Go version](https://img.shields.io/badge/Go-1.25.3-00ADD8?logo=go\&logoColor=white)
![Issues](https://img.shields.io/github/issues/bruli/raspberryWaterSystem)
[![codecov](https://codecov.io/gh/bruli/pinger/branch/main/graph/badge.svg)](https://codecov.io/gh/bruli/raspberryWaterSystem)

## Overview

RaspberryWaterSystem is a smart irrigation controller designed for bonsai trees and delicate plants.

The project runs on a Raspberry Pi and manages irrigation zones, fertilization systems, and automated watering tasks while integrating with external monitoring and automation platforms.

The main goal of the project is to provide a reliable and intelligent irrigation system focused on:

* Plant health.
* Water efficiency.
* Automation.
* Remote management.
* Observability.
* Scalability.

---

## Features

### Irrigation Control

* Multi-zone irrigation support.
* Manual and automatic watering.
* Fertilizer injection support.
* Cleaning and flushing cycles.
* Configurable watering durations.

### Automation

* Scheduled irrigation tasks.
* Smart watering integrations.
* Remote execution.
* External trigger support.

### Monitoring

* System health monitoring.
* Structured logging.
* Notifications and alerts.
* Metrics and observability support.

### Notifications

* Irrigation alerts.
* Error notifications.
* Fertilization events.
* System status updates.

---

## Use Cases

RaspberryWaterSystem is designed for:

* Bonsai collections.
* Greenhouses.
* Indoor gardens.
* Small automated irrigation systems.
* Experimental agriculture projects.

---

## Hardware

Typical installations may include:

* Raspberry Pi.
* Relay modules.
* Solenoid valves.
* Peristaltic pumps.
* Water pumps.
* Soil moisture sensors.
* Temperature sensors.

---

## Installation

### Requirements

* Raspberry Pi.
* Go 1.25+.
* Linux-based operating system.

### Clone repository

```bash
git clone https://github.com/bruli/raspberryWaterSystem.git
cd raspberryWaterSystem
```

### Install dependencies

```bash
go mod download
```

### Run

```bash
go run ./cmd/server
```

---

## Configuration

The application can be configured using environment variables.

Example:

```bash
HTTP_PORT=8080
LOG_LEVEL=info
```

---

## Docker

### Build image

```bash
docker build -t raspberry-water-system .
```

### Run container

```bash
docker run raspberry-water-system
```

---

## Development

### Run tests

```bash
go test ./...
```

### Run linter

```bash
golangci-lint run
```

### Check vulnerabilities

```bash
go tool govulncheck ./...
```

---

## Roadmap

* Smart irrigation improvements.
* Enhanced notification system.
* Additional integrations.
* Dashboard and monitoring improvements.
* Improved automation workflows.
* Extended hardware compatibility.

---

## Security

Recommendations:

* Use private networks whenever possible.
* Protect remote endpoints.
* Restrict access to management interfaces.
* Use HTTPS when exposing services.

---

## License

MIT

---

## Author

Developed by Pablo Braulio
