package runtime

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/groundhogchain/chuckd"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// PprofConfig controls live profiling and endpoints reserved by file profiling.
type PprofConfig struct {
	Enabled            bool
	CPUProfileActive   bool
	TraceProfileActive bool
}

// SetupTelemetry initializes OpenTelemetry with Prometheus and returns a function to shut it down.
func SetupTelemetry(ctx context.Context, instanceID string, addr string, config PprofConfig) (shutdown func(context.Context) error, err error) {

	rsc, err := createResource(instanceID)
	if err != nil {
		return nil, err
	}

	// Create a Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	// Create a meter provider with the Prometheus exporter
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(rsc),
	)

	// Set the global meter provider
	otel.SetMeterProvider(meterProvider)

	// Set up HTTP server to expose Prometheus metrics endpoint
	mux := telemetryMux(config)
	if config.Enabled && config.CPUProfileActive {
		liblog.Infof(ctx, "pprof /debug/pprof/profile unavailable: --cpu file profiling is active")
	}
	if config.Enabled && config.TraceProfileActive {
		liblog.Infof(ctx, "pprof /debug/pprof/trace unavailable: --trace file profiling is active")
	}
	go func() {
		liblog.Infof(ctx, "prometheus metrics endpoint started at %s/metrics", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			liblog.Fatalf(ctx, "starting metrics server: %v", err)
		}
	}()

	// Return a shutdown function that cleans up resources
	return meterProvider.Shutdown, nil
}

func telemetryMux(config PprofConfig) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	if config.Enabled {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", http.NotFound)
		if config.CPUProfileActive {
			mux.HandleFunc("/debug/pprof/profile", profileConflict("--cpu"))
		} else {
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		}
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		if config.TraceProfileActive {
			mux.HandleFunc("/debug/pprof/trace", profileConflict("--trace"))
		} else {
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		}
		for _, name := range []string{"heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
			mux.Handle("/debug/pprof/"+name, pprof.Handler(name))
		}
	}
	return mux
}

func profileConflict(flag string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "endpoint unavailable while "+flag+" file profiling is active", http.StatusConflict)
	}
}

func createResource(instanceID string) (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			// Service metadata
			semconv.ServiceNameKey.String("chuckd"),
			semconv.ServiceVersionKey.String(chuckd.Short()),
			semconv.ServiceInstanceIDKey.String(instanceID),
			// Environment information
			// attribute.String("environment", "production"),
			// attribute.String("region", "us-west-2"),
			// attribute.String("deployment.id", "deployment-123"),
			// Resource limits & utilization context
			// attribute.Int("container.memory.limit", 1024),
			// attribute.Int("container.cpu.limit", 4),
		),
	)
}
