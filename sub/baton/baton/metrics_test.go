package baton

import (
	"context"
	"testing"

	clientprom "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func installTestMeterProvider(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()
	ctx := context.Background()
	previous := otel.GetMeterProvider()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previous)
		if err := provider.Shutdown(ctx); err != nil {
			t.Errorf("shut down meter provider: %v", err)
		}
	})
	return reader
}

func collectBatonMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var data metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &data); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	return data
}

func TestBatonMetricsRecordRestoredAndCommittedHeight(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(ctx); err != nil {
			t.Errorf("shut down meter provider: %v", err)
		}
	})

	metrics := newBatonMetricsWithMeter(ctx, provider.Meter(batonMeterName))
	metrics.recordCurrentBlockHeight(ctx, 41)
	metrics.recordCommittedBlock(ctx, 42)

	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}

	height := findInt64Gauge(t, data, "coordinator_current_block_height")
	if height != 42 {
		t.Fatalf("coordinator height = %d, want 42", height)
	}

	committed := findInt64Counter(t, data, "coordinator_committed_blocks")
	if committed != 1 {
		t.Fatalf("committed blocks = %d, want 1", committed)
	}
}

func TestBatonMetricsPrometheusNames(t *testing.T) {
	ctx := context.Background()
	registry := clientprom.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		t.Fatalf("create Prometheus exporter: %v", err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	t.Cleanup(func() {
		if err := provider.Shutdown(ctx); err != nil {
			t.Errorf("shut down meter provider: %v", err)
		}
	})

	metrics := newBatonMetricsWithMeter(ctx, provider.Meter(batonMeterName))
	metrics.recordCommittedBlock(ctx, 42)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather Prometheus metrics: %v", err)
	}
	typesByName := make(map[string]string, len(families))
	for _, family := range families {
		typesByName[family.GetName()] = family.GetType().String()
	}

	if got := typesByName["coordinator_current_block_height"]; got != "GAUGE" {
		t.Fatalf("coordinator_current_block_height type = %q, want GAUGE", got)
	}
	if got := typesByName["coordinator_committed_blocks_total"]; got != "COUNTER" {
		t.Fatalf("coordinator_committed_blocks_total type = %q, want COUNTER", got)
	}
}

func findInt64Gauge(t *testing.T, resource metricdata.ResourceMetrics, name string) int64 {
	t.Helper()
	for _, scope := range resource.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != name {
				continue
			}
			gauge, ok := candidate.Data.(metricdata.Gauge[int64])
			if !ok {
				t.Fatalf("metric %s has type %T, want int64 gauge", name, candidate.Data)
			}
			if len(gauge.DataPoints) != 1 {
				t.Fatalf("metric %s has %d data points, want 1", name, len(gauge.DataPoints))
			}
			return gauge.DataPoints[0].Value
		}
	}
	t.Fatalf("metric %s not found", name)
	return 0
}

func findInt64Counter(t *testing.T, resource metricdata.ResourceMetrics, name string) int64 {
	t.Helper()
	for _, scope := range resource.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != name {
				continue
			}
			counter, ok := candidate.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("metric %s has type %T, want int64 sum", name, candidate.Data)
			}
			if !counter.IsMonotonic {
				t.Fatalf("metric %s is not monotonic", name)
			}
			if len(counter.DataPoints) != 1 {
				t.Fatalf("metric %s has %d data points, want 1", name, len(counter.DataPoints))
			}
			return counter.DataPoints[0].Value
		}
	}
	t.Fatalf("metric %s not found", name)
	return 0
}
