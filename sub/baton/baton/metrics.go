package baton

import (
	"context"

	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const batonMeterName = "github.com/groundhogchain/chuckd/sub/baton"

type batonMetrics struct {
	currentBlockHeight metric.Int64Gauge
	committedBlocks    metric.Int64Counter
}

func newBatonMetrics(ctx context.Context) batonMetrics {
	return newBatonMetricsWithMeter(ctx, otel.Meter(batonMeterName))
}

func newBatonMetricsWithMeter(ctx context.Context, meter metric.Meter) batonMetrics {
	currentBlockHeight, err := meter.Int64Gauge(
		"coordinator_current_block_height",
		metric.WithDescription("The height of the last block committed by the coordinator."),
	)
	must.NoError(ctx, err)

	committedBlocks, err := meter.Int64Counter(
		"coordinator_committed_blocks",
		metric.WithDescription("Blocks committed by the coordinator."),
	)
	must.NoError(ctx, err)

	return batonMetrics{
		currentBlockHeight: currentBlockHeight,
		committedBlocks:    committedBlocks,
	}
}

func (m batonMetrics) recordCurrentBlockHeight(ctx context.Context, height proto.BlockHeight) {
	m.currentBlockHeight.Record(ctx, height)
}

func (m batonMetrics) recordCommittedBlock(ctx context.Context, height proto.BlockHeight) {
	m.recordCurrentBlockHeight(ctx, height)
	m.committedBlocks.Add(ctx, 1)
}
