# GridBeat

[![GitHub Release](https://img.shields.io/github/release/fluxionwatt/gridbeat?color=brightgreen&label=Release)](https://github.com/fluxionwatt/gridbeat/releases)

English | [简体中文](./README-CN.md)

GridBeat is an open-source SCADA (Supervisory Control and Data Acquisition) software designed for energy sectors such as photovoltaics (PV) and energy storage. Serving as a data aggregation hub, it collects, converts, stores, and manages sensor data reported from various devices—including PV inverters, Power Conversion Systems (PCS), Energy Storage Systems (ESS), environmental monitors, smart meters, box-type transformers, and EV charging piles. It enables efficient monitoring and maintenance through a web interface and integrated APIs.

### Key Functions

- Data Aggregation: Collects real-time data from sensors in inverters, energy storage units, and other components.
- Multi-Protocol Inbound Support: Supports industrial protocols including Modbus-TCP and Modbus-RTU.
- Northbound Protocol Support: Supports protocols such as IEC104, MQTT, Modbus, and GOOSE for seamless integration with third-party management systems.
- Edge-Native Performance: Features real-time capabilities that leverage low-latency edge computing networks.
- Modular Design: Utilizes a loosely coupled, modular architecture, allowing for functional expansion via pluggable modules.
- HTTP API: Provides HTTP API interfaces for streamlined system integration.


## Quick Start

Default username is `root` and password is `admim`.

### Download

You can download from [Release](https://github.com/fluxionwatt/gridbeat/releases).

```bash
# Create dependency directories (config is the configuration file directory, log is the log file directory, data is the data file directory, and extra is the extended data directory).
$ mkdir -p gridbeat/{config,log,data,extra}
$ tar xvzf gridbeat-linux-amd64.tar.gz -C gridbeat/
$ cd gridbeat
$ ./gridbeat server
```

Open a web browser and navigate to `http://localhost:8080` to access the Web interface.

### Build

```bash
# Before starting, ensure that you have Go (version 1.25 or higher) and npm (version 25.X or higher) installed.
# setup build tool

brew install go-task/tap/go-task
brew install go-task

# source code
$ git clone https://github.com/fluxionwatt/gridbeat

# start build
$ cd gridbeat && task build
```

### Docker

```bash
$ docker run -d --name gridbeat -p 8080:8080 fluxionwatt/gridbeat:1.0.0
```

### [Modbus TCP Data Collection and MQTT Transmission](./docs/quick_start/quick_start.md)

## Get Involved

## Thanks list

* [github.com/simonvetter/modbus](https://github.com/simonvetter/modbus) for modbus (thanks!)
* [github.com/emqx/neuron](https://github.com/emqx/neuron) for modbus gateway design (thanks!)

## License

See [LICENSE](./LICENSE).
