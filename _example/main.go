package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/twistingmercury/middleware"
	"github.com/twistingmercury/telemetry/logging"
	"github.com/twistingmercury/telemetry/metrics"
	"github.com/twistingmercury/telemetry/tracing"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"log"
	"net/http"
	"os"
)

const (
	namespace      = "example"
	serviceName    = "test"
	serviceVersion = "0.0.1"
	environment    = "local"
)

func main() {
	// 1.Initialize the logging package.
	err := logging.Initialize(zerolog.DebugLevel, os.Stdout, serviceName, serviceVersion, environment)
	if err != nil {
		log.Panicf("failed to initialize logging: %v", err)
	}

	// 2. Initialize the metrics package.
	err = metrics.Initialize(namespace, serviceName)
	if err != nil {
		logging.Fatal(err, "failed to initialize metrics")
	}

	// 3.  Invoke `metrics.Publish()`
	metrics.Publish()

	// 4. Initialize the tracing package.
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		logging.Fatal(err, "failed to create trace exporter")
	}
	err = tracing.Initialize(traceExporter, serviceName, serviceVersion, environment)
	if err != nil {
		logging.Fatal(err, "failed to initialize tracing")
	}

	// 5. Initialize the middleare package.
	err = middleware.Initialize(metrics.Registry(), namespace, serviceName)

	// use gin.ReleaseMode in production!!!
	gin.SetMode(gin.DebugMode)

	// Create a new Gin router
	router := gin.New()

	// 6. Create a gin router and invoke `gin.Use(middleware.Telemetry())`.
	router.Use(gin.Recovery(), middleware.Telemetry())

	// Define a simple route
	router.GET("/hello", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	// Start the server
	log.Println("Server running on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		logging.Fatal(err, "failed to start server")
	}
}
