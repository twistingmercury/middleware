package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/twistingmercury/telemetry/logging"
	"github.com/twistingmercury/telemetry/metrics"
	"github.com/twistingmercury/telemetry/tracing"
	"github.com/twistingmercury/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/trace"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	namespace      = "example"
	serviceName    = "client"
	serviceVersion = "0.0.1"
	environment    = "local"
)

var (
	mtx          sync.Mutex
	routines     = flag.Int("goroutines", 3, "number of goroutines, default is 3")
	totalCalls   *prometheus.CounterVec
	callDuration *prometheus.HistogramVec
)

func init() {
	labels := []string{"path", "status_code", "is_error"}
	totalCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      fmt.Sprintf("%s_%s_total_api_calls", namespace, serviceName),
		Help:      "The count of all call to the API, grouped by path, http method, and status code"},
		labels)

	callDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      fmt.Sprintf("%s_%s_call_api_call_duration", namespace, serviceName),
		Help:      "The duration in milliseconds calls to the API, grouped by path, http method, and status code",
		Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5)},
		labels)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	utils.ListenForInterrupt(cancel)
	flag.Parse()

	// 1.Initialize the logging package.
	if err := logging.Initialize(zerolog.WarnLevel, os.Stdout, serviceName, serviceVersion, environment); err != nil {
		log.Panicf("failed to initialize client logging: %v", err)
	}

	// 2. initialize metrics
	if err := metrics.InitializeWithPort(ctx, "9092", namespace, serviceName); err != nil {
		logging.Fatal(err, "failed to initialize client metrics")
	}

	// 3. register the metrics to be exposed
	metrics.RegisterMetrics(totalCalls, callDuration)

	// 4.  publish the metrics
	metrics.Publish()

	// 5. initialize tracing
	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		logging.Fatal(err, "failed to create client trace exporter")
	}
	if err := tracing.Initialize(traceExporter, serviceName, serviceVersion, environment); err != nil {
		logging.Fatal(err, "failed to initialize client tracing")
	}

	for i := 0; i < *routines; i++ {
		go callEpochAPI(ctx)
	}
	logging.Info("client has started.")
	<-ctx.Done()
}

func callEpochAPI(context context.Context) {
	var ctx = context
	for {
		select {
		case <-ctx.Done():
			return
		default:
			func() {
				var err error
				var statusCode string
				var duration time.Duration
				pCtx, span := tracing.Start(ctx, "callEpochAPI", trace.SpanKindClient)
				defer func() {
					span.End()
					path := "server:8080/epoch"
					isErr := fmt.Sprintf("%v", err != nil)
					elapsed := float64(duration.Milliseconds())
					totalCalls.WithLabelValues(path, statusCode, isErr).Inc()
					callDuration.WithLabelValues(path, statusCode, isErr).Observe(elapsed)
				}()

				req, err := http.NewRequestWithContext(pCtx, "GET", "http://server:8080/epoch", nil)
				if err != nil {
					span.SetStatus(codes.Error, "error creating request")
					span.RecordError(err)
					statusCode = "n/a"
					log.Printf("error creating request: %s\n", err)
					return
				}

				client := &http.Client{
					Transport: otelhttp.NewTransport(http.DefaultTransport),
					Timeout:   time.Millisecond * 500,
				}

				start := time.Now()
				response, err := client.Do(req)
				duration = time.Since(start)
				if err != nil {
					span.SetStatus(codes.Error, "error calling server")
					statusCode = "n/a"
					span.RecordError(err)
					log.Printf("error executing request: %s\n", err)
					return
				}

				if response.StatusCode != http.StatusOK {
					span.SetStatus(codes.Error, "unexepected status code")
					statusCode = fmt.Sprintf("%d", response.StatusCode)
					log.Printf("error response: %s\n", response.Status)
					return
				}
				span.SetStatus(codes.Ok, "ok")
				time.Sleep(time.Millisecond * time.Duration(randomInt(pCtx)))
			}()
		}
	}
}

func randomInt(pCtx context.Context) int {
	_, span := tracing.Start(pCtx, "randomize", trace.SpanKindInternal)
	defer func() {
		span.SetStatus(codes.Ok, "ok")
		span.End()
	}()
	mtx.Lock()
	defer mtx.Unlock()
	low := 500
	high := 1500
	return rand.Intn(high-low+1) + low
}
