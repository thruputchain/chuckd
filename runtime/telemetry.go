package runtime

import (
	"context"
	"net/http"

	"github.com/groundhogchain/chuckd"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupTelemetry initializes OpenTelemetry with Prometheus and returns a function to shut it down
func SetupTelemetry(ctx context.Context, instanceID string, addr string) (shutdown func(context.Context) error, err error) {

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
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		liblog.Infof("prometheus metrics endpoint started at %s/metrics", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			liblog.Fatalf("starting metrics server: %v", err)
		}
	}()

	// Return a shutdown function that cleans up resources
	return meterProvider.Shutdown, nil
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
