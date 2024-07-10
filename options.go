package middleware

import "strings"

type TracingOptions struct {
	ExcludedPaths map[string]struct{}
}

type TracingOption func(*TracingOptions)

func WithExcludedPaths(paths []string) TracingOption {
	return func(opt *TracingOptions) {
		if opt.ExcludedPaths == nil {
			opt.ExcludedPaths = make(map[string]struct{})
		}

		for _, v := range paths {
			opt.ExcludedPaths[strings.ToLower(v)] = struct{}{}
		}
	}
}

func MakeTracingOptions(opts ...TracingOption) TracingOptions {
	opt := TracingOptions{}

	for _, apply := range opts {
		apply(&opt)
	}

	return opt
}

type MetricsOptions struct {
	// for future use
}

type MetricsOption func(*MetricsOptions)

type LoggingOptions struct {
	// for future use
}

type LoggingOption func(*LoggingOptions)
