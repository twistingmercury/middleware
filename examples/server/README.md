# Example: Gin Middleware

This example demonstrates how to use the Gin Middleware package to instrument and trace incoming requests in a Gin web application. The middleware integrates with the OpenTelemetry framework through the [github.com/twistingmercury/telemetry](https://github.com/twistingmercury/telemetry/blob/main/readme.md) to collect and export telemetry data, including metrics and traces.

## Prerequisites

Before running the example, ensure that you have the following prerequisites installed:

- Go programming language (serviceVersion 1.21 or higher)
- Gin web framework (`go get -u github.com/gin-gonic/gin`)
- Telemetry packages:
    - `go get github.com/twistingmercury/telemetry`
    - `go get github.com/twistingmercury/middleware`

## Usage

From a terminal, follow these steps to run the example:

1. Run the example:

   ```bash
   make all
   ```

   The server will start running on `http://localhost:8080`.

2. Access the `/hello` route in your web browser or using a tool like cURL:

   ```bash
   curl http://localhost:8080/hello
   ```
   You should see the response "Hello, World!".

## Tools

### View telemetry locally

* To view the traces, use [Jaeger](http://localhost:16686)
* To view the logs, observe the console
* To view metrics, use [Grafana](http://localhost:3000)