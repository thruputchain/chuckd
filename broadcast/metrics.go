package broadcast

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/must"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type BroadcastMetrics struct {
	NumSent           metric.Int64Counter
	NumReceived       metric.Int64Counter
	NumNotReceived    metric.Int64Counter
	SendThroughput    metric.Int64Counter
	ReceiveThroughput metric.Int64Counter
}

func NewBroadcastMetrics(ctx context.Context) *BroadcastMetrics {
	m := otel.Meter("groundhogchain/chuckd")
	b := &BroadcastMetrics{}
	var err error
	b.NumSent, err = m.Int64Counter("num_sent")
	must.NoError(ctx, err)
	b.NumReceived, err = m.Int64Counter("num_received")
	must.NoError(ctx, err)
	b.NumNotReceived, err = m.Int64Counter("num_not_received")
	must.NoError(ctx, err)
	b.SendThroughput, err = m.Int64Counter("send_throughput")
	must.NoError(ctx, err)
	b.ReceiveThroughput, err = m.Int64Counter("receive_throughput")
	must.NoError(ctx, err)
	return b
}

func (b *BroadcastMetrics) DeliveryQueueMetrics() *DeliveryQueueMetrics {
	return &DeliveryQueueMetrics{
		NumSent:        b.NumSent,
		SendThroughput: b.SendThroughput,
	}
}

type DeliveryQueueMetrics struct {
	NumSent        metric.Int64Counter
	SendThroughput metric.Int64Counter
}
