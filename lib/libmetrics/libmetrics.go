package libmetrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type ctxMetricAttributes struct{}

func WithMetricAttributes(ctx context.Context, kv ...attribute.KeyValue) context.Context {
	attrs, _ := ctx.Value(ctxMetricAttributes{}).([]attribute.KeyValue)
	x := make([]attribute.KeyValue, len(attrs))
	copy(x, attrs)
	x = append(x, kv...)
	return context.WithValue(ctx, ctxMetricAttributes{}, x)
}

func WithCtxAttributes(ctx context.Context) metric.MeasurementOption {
	attrs, _ := ctx.Value(ctxMetricAttributes{}).([]attribute.KeyValue)
	return metric.WithAttributes(attrs...)
}
