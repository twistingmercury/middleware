package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/twistingmercury/middleware/v2"
	"github.com/twistingmercury/telemetry/v2/logging"
	"github.com/twistingmercury/telemetry/v2/metrics"
	"github.com/twistingmercury/telemetry/v2/tracing"
	"net/http"
	"net/http/httptest"
	"testing"

	gonic "github.com/gin-gonic/gin"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

const (
	namespace      = "unit"
	serviceName    = "test"
	serviceVersion = "1.0.0"
	environment    = "test"
)

var (
	lbuffer *bytes.Buffer
	tbuffer *bytes.Buffer
)

func initializeTests(t *testing.T) {
	lbuffer = &bytes.Buffer{}
	err := logging.Initialize(zerolog.DebugLevel, lbuffer, serviceName, serviceVersion, environment)
	require.NoError(t, err)
	defer resetTests()

	tbuffer = &bytes.Buffer{}
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint(), stdouttrace.WithWriter(tbuffer))
	err = tracing.Initialize(traceExporter, serviceName, serviceVersion, environment)
	require.NoError(t, err)

	err = metrics.Initialize(context.TODO(), namespace, serviceName)
	require.NoError(t, err)
}

func resetTests() {
	lbuffer.Reset()
	tbuffer.Reset()
}

func TestPrometheusMetricsWithEmptyRegistry(t *testing.T) {
	var registry *prometheus.Registry
	assert.Panics(t, func() { middleware.PrometheusMetrics(registry, "namespace", serviceName) })
}

func TestPrometheusMetricsWithEmptyNamespace(t *testing.T) {
	var registry = new(prometheus.Registry)
	assert.Panics(t, func() { middleware.PrometheusMetrics(registry, "", serviceName) })
}

func TestPrometheusMetricsWithEmptyServiceName(t *testing.T) {
	var registry = new(prometheus.Registry)
	assert.Panics(t, func() { middleware.PrometheusMetrics(registry, "namespace", "") })
}

func TestGinOTelMiddleware(t *testing.T) {
	initializeTests(t)
	defer resetTests()

	gonic.SetMode(gonic.TestMode)
	r := gonic.New()

	r.Use(
		middleware.PrometheusMetrics(metrics.Registry(), namespace, serviceName),
		middleware.OtelTracing(),
		middleware.Logging())
	r.GET("/test", func(c *gonic.Context) {
		c.Status(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	checkLog(t, "info", true)
}

func TestGinOTelMiddlewareLogging(t *testing.T) {
	initializeTests(t)
	defer resetTests()

	gonic.SetMode(gonic.TestMode)
	r := gonic.New()

	r.Use(middleware.Logging())
	r.GET("/test", func(c *gonic.Context) {
		c.Status(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	checkLog(t, "info", false)
}

func TestGinOTelMiddlewareInternalServerError(t *testing.T) {
	initializeTests(t)
	defer resetTests()

	gonic.SetMode(gonic.TestMode)
	r := gonic.New()
	r.Use(
		middleware.PrometheusMetrics(metrics.Registry(), namespace, serviceName),
		middleware.OtelTracing(),
		middleware.Logging())

	r.GET("/test", func(c *gonic.Context) {
		c.Errors = []*gonic.Error{
			{
				Err:  errors.New("test error"),
				Type: 0,
				Meta: nil,
			}}
		c.Status(http.StatusInternalServerError)
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	checkLog(t, "error", true)
}

func TestGinOTelMiddlewareTraceExcludedPath(t *testing.T) {
	initializeTests(t)
	defer resetTests()

	gonic.SetMode(gonic.TestMode)
	r := gonic.New()
	r.Use(middleware.OtelTracing("/test"))
	r.GET("/test", func(c *gonic.Context) {
		c.Status(http.StatusOK)
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	logText := lbuffer.String()
	require.Empty(t, logText)
}

func checkLog(t *testing.T, expectedLogLevel string, expectTracingAttrs bool) {
	logText := lbuffer.String()
	require.NotEmptyf(t, logText, "no logs found")

	var logEntry map[string]any

	err := json.Unmarshal([]byte(logText), &logEntry)
	require.NoError(t, err)

	if expectTracingAttrs {
		require.Contains(t, logEntry, "otel.trace_id")
		require.Len(t, logEntry["otel.trace_id"], 32)
		require.NotEqual(t, oteltrace.TraceID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, logEntry["otel.trace_id"])

		require.Contains(t, logEntry, "otel.span_id")
		require.Len(t, logEntry["otel.span_id"], 16)
		require.NotEqual(t, oteltrace.SpanID{0, 0, 0, 0, 0, 0, 0, 0}, logEntry["otel.trace_id"])
	} else {
		require.NotContains(t, logEntry, "otel.trace_id")
		require.NotContains(t, logEntry, "otel.span_id")
	}

	require.Equal(t, expectedLogLevel, logEntry["level"])
	require.Equal(t, serviceName, logEntry["service"])
	require.Equal(t, serviceVersion, logEntry["version"])
	require.Equal(t, environment, logEntry["environment"])
}

func TestSpanStatus(t *testing.T) {
	testCases := []struct {
		name     string
		status   int
		expected struct {
			code codes.Code
			desc string
		}
	}{
		{
			name:   "OK",
			status: http.StatusOK,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Ok, desc: "OK"},
		},
		{
			name:   "Bad Request",
			status: http.StatusBadRequest,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Ok, desc: "Bad Request"},
		},
		{
			name:   "Unauthorized",
			status: http.StatusUnauthorized,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Ok, desc: "Unauthorized"},
		},
		{
			name:   "Forbidden",
			status: http.StatusForbidden,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Ok, desc: "Forbidden"},
		},
		{
			name:   "Not Found",
			status: http.StatusNotFound,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Ok, desc: "Not Found"},
		},
		{
			name:   "Method Not Allowed",
			status: http.StatusMethodNotAllowed,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Ok, desc: "Method Not Allowed"},
		},
		{
			name:   "Internal Server Error",
			status: http.StatusInternalServerError,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Error, desc: "Internal Server Error"},
		},
		{
			name:   "Bad Gateway",
			status: http.StatusBadGateway,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Error, desc: "Bad Gateway"},
		},
		{
			name:   "Service Unavailable",
			status: http.StatusServiceUnavailable,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Error, desc: "Service Unavailable"},
		},
		{
			name:   "Unknown",
			status: http.StatusTeapot,
			expected: struct {
				code codes.Code
				desc string
			}{code: codes.Unset, desc: ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, desc := middleware.SpanStatus(tc.status)
			assert.Equal(t, tc.expected.code, code)
			assert.Equal(t, tc.expected.desc, desc)
		})
	}
}

type testUserAgent struct {
	ua      string
	uaType  string
	browser string
}

const nilValue = "<nil>"

var testUserAgents = []testUserAgent{
	{"Mozilla/5.0 (Linux; Android 7.0; SM-T827R4 Build/NRD90M) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.116 Safari/537.36", "mobile", middleware.BrowserChrome},
	{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; Yahoo! Slurp; http://help.yahoo.com/help/us/ysearch/slurp)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)", "bot", nilValue},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36 Edge/12.246", "desktop", middleware.BrowserEdge},
	{"Mozilla/5.0 (iPhone13,2; U; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/602.1.50 (KHTML, like Gecko) Version/10.0 Mobile/15E148 Safari/602.1", "mobile", middleware.BrowserSafari},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/601.3.9 (KHTML, like Gecko) Version/9.0.2 Safari/601.3.9", "desktop", middleware.BrowserSafari},
	{"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:15.0) Gecko/20100101 Firefox/15.0.1", "desktop", middleware.BrowserFirefox},
	{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36 OPR/102.0.0.0", "desktop", middleware.BrowserOpera},
	{"Mozilla/5.0 (Windows NT 10.0; Trident/7.0; rv:11.0) like Gecko", "desktop", middleware.BrowserIE},
	{"", "desktop", middleware.BrowserIE},
}

func TestParseUserAgent(t *testing.T) {
	for _, tua := range testUserAgents {
		kvps := make(map[any]any)
		raw := middleware.ParseUserAgent(tua.ua)
		for k, v := range raw {
			kvps[k] = v
		}

		if len(tua.ua) > 0 {
			actual := kvps[middleware.UserAgentDevice]
			assert.Equal(t, tua.uaType, actual)
		}
	}
}

func TestParseHeaders(t *testing.T) {
	testCases := []struct {
		name           string
		headers        map[string][]string
		expectedResult map[string]any
	}{
		{
			name: "Single header",
			headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expectedResult: map[string]any{
				"http.content-type": "application/json",
			},
		},
		{
			name: "Multiple headers",
			headers: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer token123"},
				"Accept":        {"application/json", "text/plain"},
			},
			expectedResult: map[string]any{
				"http.content-type":  "application/json",
				"http.authorization": "bearer token123",
				"http.accept":        "application/json, text/plain",
			},
		},
		{
			name:           "Empty headers",
			headers:        map[string][]string{},
			expectedResult: map[string]any{},
		},
		{
			name: "Header with multiple values",
			headers: map[string][]string{
				"Accept": {"application/json", "text/plain", "application/xml"},
			},
			expectedResult: map[string]any{
				"http.accept": "application/json, text/plain, application/xml",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := middleware.ParseHeaders(tc.headers)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestContainsPath(t *testing.T) {
	exclusions := []string{"/test/1", "/test/3", "/test/5"}
	inclusions := []string{"/test/2", "/test/4", "/test/6"}

	for _, ex := range exclusions {
		result := middleware.ContainsPath(exclusions, ex)
		assert.True(t, result)
	}

	for _, in := range inclusions {
		result := middleware.ContainsPath(exclusions, in)
		assert.False(t, result)
	}
}

func TestNormalize(t *testing.T) {
	type testCase struct {
		target   string
		expected string
	}
	const expected = "this_is_a_target"
	testCases := []testCase{
		{"this  is a TARGET", expected},
		{"this-is-a:target", expected},
		{"this::is::a-target", expected},
	}
	for _, tc := range testCases {
		results := middleware.Normalize(tc.target)
		assert.Equal(t, tc.expected, results)
	}
}

func TestFromMap(t *testing.T) {
	target := make(map[string]any)
	target["key1"] = "value1"
	target["fruit"] = "orange"
	target["color"] = "blue"

	expected := make([]logging.KeyValue, len(target))
	expected[0] = logging.KeyValue{Key: "key1", Value: "value1"}
	expected[1] = logging.KeyValue{Key: "fruit", Value: "orange"}
	expected[2] = logging.KeyValue{Key: "color", Value: "blue"}

	results := middleware.FromMap(target)
	assert.ElementsMatch(t, expected, results)

}
