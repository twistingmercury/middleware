# middleware Package

The twistingmercury/middleware package is a Go library that provides middleware for instrumenting and tracing incoming requests in a Gin web application. It uses [twistingmercury/telemetry](https://github.com/twistingmercury/telemetry) to collect and export telemetry data, including metrics and traces.

## Features

- Middleware for instrumenting incoming requests in a [gin-gonic/gin](https://github.com/gin-gonic/gin) web application
- Integration with OpenTelemetry for collecting and exporting traces
- Automatic generation of request duration and count metrics
- Parsing of user agent and request headers for detailed telemetry data
- Customizable span naming and attribute generation
- Correlation of logs with trace and span information

## Installation

```bash
go get github.com/twistingmercury/telemetry/middleware
```

## Usage

For a working example, view and run the example: [_example/main.go](_example/main.go). In general, take the following steps:

1. [Initialize the logging package.
2. Initialize the metrics package.
3. Invoke `metrics.Publish()`
4. Initialize the tracing package.
5. Initialize the middleare package.
6. Create a gin router and invoke `gin.Use(middleware.Telemetry())`.

After that, you can define your routes and handlers as usual, and the middleware will automatically instrument and trace the incoming requests.

## Telemetry Data

The Gin Middleware package generates the following telemetry data for the gin handlers (RESTful endpoints):

- Request duration metric: Measures the duration of each incoming request in milliseconds.
- Request count metric: Counts the number of incoming requests.
- Request trace: Creates a trace for each incoming request, including span information.
- Detailed request information: Parses the user agent and request headers to include additional attributes in the telemetry data.

### Telemetry collection

* Logs: Logs are written to stdout.
* Traces: Traces are sent to the configured exporter. In a production environment, you'd create gRPC exporter and send the data to an OTel collector.
* Metrics: Metrics are exposed by default over http on port 9090, i.e., `http://[my-api]:9090/metrics`

## Contributing

Contributions to the twistingmercury/telemetry package are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request on the GitHub repository.

## License

The twistingmercury/middleware package is open-source and released under the [MIT License](./LICENSE).