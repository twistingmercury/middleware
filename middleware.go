package middleware

import (
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/twistingmercury/telemetry/v2/logging"
	"github.com/twistingmercury/telemetry/v2/tracing"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mileusna/useragent"

	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const ( // for metric vectors
	methodLabel = "http_method"
	statusLabel = "http_status"
	pathLabel   = "http_path"
)

const ( // for user agent properties and values
	UserAgentOS             = "http.user_agent.os"
	UserAgentOSVersion      = "http.user_agent.os_version"
	UserAgentDevice         = "http.user_agent.device"
	UserAgentBrowser        = "http.user_agent.browser"
	UserAgentBrowserVersion = "http.user_agent.browser_version"
	BrowserChrome           = "chrome"
	BrowserSafari           = "safari"
	BrowserFirefox          = "firefox"
	BrowserOpera            = "opera"
	BrowserIE               = "ie"
	BrowserEdge             = "edge"
	BrowserTrident          = "Trident"
	DeviceMobile            = "mobile"
	DeviceDesktop           = "desktop"
	DeviceBot               = "bot"
)

const ( // for http request properties and header values
	Http            = "http"
	Https           = "https"
	HttpMethod      = "http.request.method"
	HttpPath        = "http.request.path"
	HttpRemoteAddr  = "http.request.remoteAddr"
	HttpRequestHost = "http.request.host"
	HttpStatus      = "http.response.status"
	HttpLatency     = "http.response.latency"
	TLSVersion      = "http.tls.serviceVersion"
	HttpScheme      = "http.scheme"

	//QueryString = "http.request.queryString"
) //

var (
	reg             *prometheus.Registry
	totalCalls      *prometheus.CounterVec
	concurrentCalls *prometheus.GaugeVec
	callDuration    *prometheus.HistogramVec
	apiName         string
	nspace          string
)

//// Initialize preps the middleware.
////
//// Deprecated: use GetMetricsMiddleware, GetTracingMiddleware, and GetLoggingMiddleware which do not require separate initialization.
//func Initialize(registry *prometheus.Registry, namespace, apiname string) error {
//	switch {
//	case registry == nil:
//		return errors.New("registry is nil")
//	case strings.TrimSpace(namespace) == "":
//		return errors.New("namespace is empty")
//	case strings.TrimSpace(apiname) == "":
//		return errors.New("apiname is empty")
//	}
//
//	reg = registry
//	nspace = namespace
//	apiName = apiname
//	concurrentCalls, totalCalls, callDuration = Metrics()
//	reg.MustRegister(concurrentCalls, totalCalls, callDuration)
//	return nil
//}

// Telemetry returns middleware that will instrument and trace incoming requests.
//
// Deprecated: use GetMetricsMiddleware, GetTracingMiddleware, and GetLoggingMiddleware to get separated middleware for metrics, tracing, and logging.
//func Telemetry() gin.HandlerFunc {
//	return func(c *gin.Context) {
//		path := c.Request.URL.Path
//		method := c.Request.Method
//		var elapsedTime float64
//		var statusCode string
//		concurrentCalls.WithLabelValues(path, method).Inc()
//		defer func() {
//			concurrentCalls.WithLabelValues(path, method).Dec()
//			callDuration.WithLabelValues(path, method, statusCode).Observe(elapsedTime)
//			totalCalls.WithLabelValues(path, method, statusCode).Inc()
//		}()
//
//		spanName := fmt.Sprintf("%s: %s", c.Request.Method, c.Request.URL.Path)
//		parentCtx := tracing.ExtractContext(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
//		childCtx, span := tracing.Start(parentCtx, spanName, oteltrace.SpanKindServer, semconv.HTTPRoute(spanName))
//		c.Request = c.Request.WithContext(childCtx)
//		defer span.End()
//
//		before := time.Now()
//		c.Next()
//		elapsedTime = float64(time.Since(before)) / float64(time.Millisecond)
//
//		logRequest(c, elapsedTime)
//		code, desc := SpanStatus(c.Writer.Status())
//		span.SetStatus(code, desc)
//
//		statusCode = strconv.Itoa(c.Writer.Status())
//	}
//}

// Logging returns the logging middleware
func Logging(excludePaths ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if containsPath(excludePaths, path) {
			c.Next()
			return
		}
		var elapsedTime float64

		before := time.Now()
		c.Next()
		elapsedTime = float64(time.Since(before)) / float64(time.Millisecond)

		logRequest(c, elapsedTime)
	}
}

// PrometheusMetrics returns the metrics middleware used by the Prometheus software.
func PrometheusMetrics(registry *prometheus.Registry, namespace string, apiname string, excludePaths ...string) gin.HandlerFunc {
	switch {
	case registry == nil:
		panic("registry is nil")
	case strings.TrimSpace(namespace) == "":
		panic("namespace is empty")
	case strings.TrimSpace(apiname) == "":
		panic("apiname is empty")
	}

	reg = registry
	nspace = namespace
	apiName = apiname

	concurrentCalls, totalCalls, callDuration = Metrics()
	reg.MustRegister(concurrentCalls, totalCalls, callDuration)

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if containsPath(excludePaths, path) {
			c.Next()
			return
		}

		method := c.Request.Method
		var elapsedTime float64
		var statusCode string
		concurrentCalls.WithLabelValues(path, method).Inc()
		defer func() {
			concurrentCalls.WithLabelValues(path, method).Dec()
			callDuration.WithLabelValues(path, method, statusCode).Observe(elapsedTime)
			totalCalls.WithLabelValues(path, method, statusCode).Inc()
		}()

		before := time.Now()
		c.Next()
		elapsedTime = float64(time.Since(before)) / float64(time.Millisecond)
	}
}

// Metrics provides the prometheus metrics that are to be tracked.
func Metrics() (*prometheus.GaugeVec, *prometheus.CounterVec, *prometheus.HistogramVec) {
	concurrentCallsName := normalize(fmt.Sprintf("%s_concurrent_calls", apiName))
	concurrentCalls := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: nspace,
		Name:      concurrentCallsName,
		Help:      "the count of concurrent calls to the APIs, grouped by path and http method"},
		[]string{pathLabel, methodLabel})

	totalCallsName := normalize(fmt.Sprintf("%s_total_calls", apiName))
	totalCalls := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: nspace,
		Name:      totalCallsName,
		Help:      "The count of all call to the API, grouped by path, http method, and status code"},
		[]string{pathLabel, methodLabel, statusLabel})

	callDurationName := normalize(fmt.Sprintf("%s_call_duration", apiName))
	callDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: nspace,
		Name:      callDurationName,
		Help:      "The duration in milliseconds calls to the API, grouped by path, http method, and status code",
		Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5)},
		[]string{pathLabel, methodLabel, statusLabel})

	return concurrentCalls, totalCalls, callDuration
}

// OtelTracing returns the tracing middleware.
func OtelTracing(excludePaths ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if containsPath(excludePaths, path) {
			c.Next()
			return
		}

		spanName := fmt.Sprintf("%s: %s", c.Request.Method, c.Request.URL.Path)
		parentCtx := tracing.ExtractContext(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		childCtx, span := tracing.Start(parentCtx, spanName, oteltrace.SpanKindServer, semconv.HTTPRoute(spanName))
		c.Request = c.Request.WithContext(childCtx)
		defer span.End()

		c.Next()

		code, desc := SpanStatus(c.Writer.Status())
		span.SetStatus(code, desc)
	}
}

// SpanStatus returns the OpenTelemetry statusLabel code as defined in
// go.opentelemetry.io/old_elemetry/codes and a brief description for a given HTTP statusLabel code.
func SpanStatus(status int) (code otelCodes.Code, desc string) {
	switch {
	case status >= 200 && status < 300:
		code = otelCodes.Ok
		desc = "OK"
	case status == 400:
		code = otelCodes.Ok
		desc = "Bad Request"
	case status == 401:
		code = otelCodes.Ok
		desc = "Unauthorized"
	case status == 403:
		code = otelCodes.Ok
		desc = "Forbidden"
	case status == 404:
		code = otelCodes.Ok
		desc = "Not Found"
	case status == 405:
		code = otelCodes.Ok
		desc = "Method Not Allowed"
	case status == 500:
		code = otelCodes.Error
		desc = "Internal Server Error"
	case status == 502:
		code = otelCodes.Error
		desc = "Bad Gateway"
	case status == 503:
		code = otelCodes.Error
		desc = "Service Unavailable"
	default:
		code = otelCodes.Unset
		desc = ""
	}
	return
}

func logRequest(c *gin.Context, elapsedTime float64) {
	ctx := c.Request.Context()
	defer func() {
		if r := recover(); r != nil {
			logging.Error(ctx, errors.New("panic in logging middleware"),
				"panic in logging middleware", logging.KeyValue{Key: "panic", Value: r})
		}
	}()

	status := c.Writer.Status()
	args := map[string]any{
		HttpMethod:     c.Request.Method,
		HttpPath:       c.Request.URL.Path,
		HttpRemoteAddr: c.Request.RemoteAddr,
		HttpStatus:     status,
		HttpLatency:    fmt.Sprintf("%fms", elapsedTime),
	}

	scheme := Http
	if c.Request.TLS != nil {
		scheme = Https
		args[TLSVersion] = c.Request.TLS.Version
	}

	args[HttpScheme] = scheme
	args[HttpRequestHost] = c.Request.Host

	/* !!! this could log sensitive data. leaving out for now. !!!
	if rQuery := c.Request.URL.RawQuery; len(rQuery) > 0 {
		args[QueryString] = rQuery
	}
	*/

	hd := ParseHeaders(c.Request.Header)
	args = logging.MergeMaps(args, hd)
	ua := ParseUserAgent(c.Request.UserAgent())
	args = logging.MergeMaps(args, ua)

	logAttribs := fromMap(args)
	if status > 499 || c.Errors.Last() != nil {
		errs := strings.Join(c.Errors.Errors(), ";")
		logging.Error(ctx, errors.New(errs), "request failed", logAttribs...)
		return
	}

	logging.Info(ctx, "request successful", logAttribs...)
}

// ParseHeaders parses the headers and returns a map of attribs.
func ParseHeaders(headers map[string][]string) (args map[string]any) {
	args = make(map[string]any)
	for k, v := range headers {
		args[strings.ToLower("http."+k)] = strings.ToLower(strings.Join(v, ", "))
	}
	return
}

// ParseUserAgent parses the user agent string and returns a map of attribs.
func ParseUserAgent(rawUserAgent string) (args map[string]any) {
	if len(rawUserAgent) == 0 {
		return //no-op
	}

	args = make(map[string]any)
	ua := useragent.Parse(rawUserAgent)

	args[UserAgentOS] = ua.OS
	args[UserAgentOSVersion] = ua.OSVersion

	var device string
	switch {
	case ua.Mobile || ua.Tablet:
		device = DeviceMobile
	case ua.Desktop:
		device = DeviceDesktop
	case ua.Bot:
		device = DeviceBot
	}

	args[UserAgentDevice] = device

	var browser string
	if ua.Mobile || ua.Tablet || ua.Desktop {
		switch {
		case ua.IsChrome():
			browser = BrowserChrome
		case ua.IsSafari():
			browser = BrowserSafari
		case ua.IsFirefox():
			browser = BrowserFirefox
		case ua.IsOpera():
			browser = BrowserOpera
		case ua.IsInternetExplorer() || strings.Contains(rawUserAgent, BrowserTrident):
			browser = BrowserIE
		case ua.IsEdge():
			browser = BrowserEdge
		}

		args[UserAgentBrowser] = browser
		args[UserAgentBrowserVersion] = ua.Version
	}
	return
}

func containsPath(paths []string, exclusion string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		if path == exclusion {
			return true
		}
	}
	return false
}

func normalize(name string) string {
	r := regexp.MustCompile(`\s+`)
	name = r.ReplaceAllString(name, "_")
	r = regexp.MustCompile(`[./:_-]`)
	return r.ReplaceAllString(strings.ToLower(name), "_")
}

func fromMap(m map[string]any) []logging.KeyValue {
	values := make([]logging.KeyValue, 0, len(m))
	for k, v := range m {
		values = append(values, logging.KeyValue{Key: k, Value: v})
	}
	return values
}
