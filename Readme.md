# Orchestra 🎭 - A Distributed Task Orchestration System

## Overview 🌟

Orchestra is a powerful distributed task orchestration system built in Go, designed to efficiently manage and execute containerized tasks across a cluster of worker nodes. It provides a robust platform for scheduling, monitoring, and controlling distributed workloads.

## Features ✨

### Distributed Task Management
- Efficiently distribute tasks across multiple worker nodes
- Docker Integration: Native support for containerized workloads
- RESTful API: Simple HTTP API for task management and monitoring
- Real-time Monitoring: Track task states and resource usage
- Flexible Scheduling: Smart task scheduling based on node resources
- State Management: Reliable task state tracking and persistence
- Resource Management: CPU, Memory, and Disk usage monitoring

## Architecture 🏗️

Orchestra follows a manager-worker architecture:

- Manager: Coordinates task distribution and maintains system state
- Workers: Execute tasks in Docker containers and report status
- API Layer: RESTful endpoints for system interaction
- Store: Flexible storage interface for task and event data

## Getting Started 🚀

### Prerequisites

- Go 1.23.3 or higher
- Docker
- Git

### Installation

```bash
# Clone the repository
git clone https://github.com/utkarsh5026/Orchestra.git

# Navigate to the project directory
cd Orchestra

# Install dependencies
go mod download

# Build the project
go build -o orchestra src/main.go
```

## Project Structure 📁

The main components are organized as follows:

- `cmd/`: Command-line interface definitions
- `manager/`: Manager node implementation
- `worker/`: Worker node implementation
- `task/`: Task definitions and Docker integration
- `store/`: Storage implementations
- `node/`: Node management and statistics
- `handler/`: HTTP request handlers