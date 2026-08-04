// Copyright 2026 Cisco Systems, Inc. and its affiliates
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/defenseclaw/defenseclaw/internal/gatewaylog"
	"github.com/defenseclaw/defenseclaw/internal/observability"
	collectormetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

func TestOTLPInboundInsightClawPromptTokensMapToCanonicalModelIOMetric(t *testing.T) {
	previousInstance := gatewaylog.SidecarInstanceID()
	gatewaylog.SetSidecarInstanceID("otlp-insightclaw-compat-token-test")
	t.Cleanup(func() { gatewaylog.SetSidecarInstanceID(previousInstance) })

	fixture := newOTLPV8MetricFixture(t)
	api := &APIServer{}
	api.bindOTLPObservabilityRuntime(fixture.runtime)
	now := time.Now().UTC()

	point := &metricspb.NumberDataPoint{
		TimeUnixNano: uint64(now.UnixNano()),
		Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("gen_ai.response.model", "gpt-4o"),
			otlpClassifierStringAttribute("openclaw.provider", "openai"),
			otlpClassifierStringAttribute("gen_ai.agent.id", "agent-insight"),
			otlpClassifierStringAttribute("gen_ai.conversation.id", "conversation-insight"),
		},
		Value: &metricspb.NumberDataPoint_AsInt{AsInt: 128},
	}
	message := &collectormetricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricspb.ResourceMetrics{{
		Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("service.name", "openclaw-gateway"),
		}},
		ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: []*metricspb.Metric{{
			Name: "openclaw.llm.tokens.prompt",
			Unit: "tokens",
			Data: &metricspb.Metric_Gauge{Gauge: &metricspb.Gauge{DataPoints: []*metricspb.NumberDataPoint{point}}},
		}}}},
	}}}

	accounting, err := api.importDecodedOTLPRequestV8(
		context.Background(), message, otelSignalMetrics, "openclaw", now,
	)
	if err != nil || accounting.derivedOnly != 1 || !accounting.valid() {
		t.Fatalf("InsightClaw prompt-token accounting=%+v err=%v", accounting, err)
	}

	metrics := fixture.pipelines.sinks(t, 1).local.snapshot()
	if len(metrics) != 1 || metrics[0].Descriptor().Name != "gen_ai.client.token.usage" {
		t.Fatalf("compat token metrics = %#v", metrics)
	}
	if metrics[0].CanonicalRecord().Bucket() != observability.BucketModelIO {
		t.Fatalf("token compatibility bucket=%q want %q", metrics[0].CanonicalRecord().Bucket(), observability.BucketModelIO)
	}
	attributes := metrics[0].Attributes()
	if attributes["gen_ai.token.type"] != "input" ||
		attributes["gen_ai.provider.name"] != "openai" ||
		attributes["gen_ai.request.model"] != "gpt-4o" {
		t.Fatalf("compat token labels = %#v", attributes)
	}
	correlation := metrics[0].CanonicalRecord().Correlation()
	if correlation.AgentID != "agent-insight" || correlation.SessionID != "conversation-insight" {
		t.Fatalf("compat token correlation = %#v", correlation)
	}
}

func TestOTLPInboundInsightClawToolCallsMapToToolActivityBucket(t *testing.T) {
	previousInstance := gatewaylog.SidecarInstanceID()
	gatewaylog.SetSidecarInstanceID("otlp-insightclaw-compat-tool-test")
	t.Cleanup(func() { gatewaylog.SetSidecarInstanceID(previousInstance) })

	fixture := newOTLPV8MetricFixture(t)
	api := &APIServer{}
	api.bindOTLPObservabilityRuntime(fixture.runtime)
	now := time.Now().UTC()

	point := &metricspb.NumberDataPoint{
		TimeUnixNano: uint64(now.UnixNano()),
		Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("tool.name", "shell"),
			otlpClassifierStringAttribute("openclaw.channel", "terminal"),
			otlpClassifierStringAttribute("gen_ai.agent.id", "agent-tool"),
			otlpClassifierStringAttribute("openclaw.session.key", "session-tool"),
		},
		Value: &metricspb.NumberDataPoint_AsInt{AsInt: 3},
	}
	message := &collectormetricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricspb.ResourceMetrics{{
		Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("service.name", "openclaw-gateway"),
		}},
		ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: []*metricspb.Metric{{
			Name: "openclaw.tool.calls",
			Unit: "calls",
			Data: &metricspb.Metric_Sum{Sum: &metricspb.Sum{
				AggregationTemporality: metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				IsMonotonic:           true,
				DataPoints:             []*metricspb.NumberDataPoint{point},
			}},
		}}}},
	}}}

	accounting, err := api.importDecodedOTLPRequestV8(
		context.Background(), message, otelSignalMetrics, "openclaw", now,
	)
	if err != nil || accounting.derivedOnly != 1 || !accounting.valid() {
		t.Fatalf("InsightClaw tool-call accounting=%+v err=%v", accounting, err)
	}

	metrics := fixture.pipelines.sinks(t, 1).local.snapshot()
	if len(metrics) != 1 || metrics[0].Descriptor().Name != "defenseclaw.tool.calls" {
		t.Fatalf("compat tool metrics = %#v", metrics)
	}
	if metrics[0].CanonicalRecord().Bucket() != observability.BucketToolActivity {
		t.Fatalf("tool compatibility bucket=%q want %q", metrics[0].CanonicalRecord().Bucket(), observability.BucketToolActivity)
	}
	attributes := metrics[0].Attributes()
	if attributes["gen_ai.tool.name"] != "shell" || attributes["tool.provider"] != "terminal" {
		t.Fatalf("compat tool labels = %#v", attributes)
	}
	correlation := metrics[0].CanonicalRecord().Correlation()
	if correlation.AgentID != "agent-tool" || correlation.SessionID != "session-tool" {
		t.Fatalf("compat tool correlation = %#v", correlation)
	}
}

func TestOTLPInboundInsightClawReportedCostMapsToAgentLifecycleBucket(t *testing.T) {
	previousInstance := gatewaylog.SidecarInstanceID()
	gatewaylog.SetSidecarInstanceID("otlp-insightclaw-compat-cost-test")
	t.Cleanup(func() { gatewaylog.SetSidecarInstanceID(previousInstance) })

	fixture := newOTLPV8MetricFixture(t)
	api := &APIServer{}
	api.bindOTLPObservabilityRuntime(fixture.runtime)
	now := time.Now().UTC()

	point := &metricspb.NumberDataPoint{
		TimeUnixNano: uint64(now.UnixNano()),
		Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("gen_ai.response.model", "gpt-4o"),
			otlpClassifierStringAttribute("openclaw.provider", "openai"),
			otlpClassifierStringAttribute("gen_ai.agent.id", "agent-cost"),
			otlpClassifierStringAttribute("openclaw.session.key", "session-cost"),
		},
		Value: &metricspb.NumberDataPoint_AsDouble{AsDouble: 1.25},
	}
	message := &collectormetricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricspb.ResourceMetrics{{
		Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("service.name", "openclaw-gateway"),
		}},
		ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: []*metricspb.Metric{{
			Name: "openclaw.cost.usd",
			Unit: "usd",
			Data: &metricspb.Metric_Gauge{Gauge: &metricspb.Gauge{DataPoints: []*metricspb.NumberDataPoint{point}}},
		}}}},
	}}}

	accounting, err := api.importDecodedOTLPRequestV8(
		context.Background(), message, otelSignalMetrics, "openclaw", now,
	)
	if err != nil || accounting.derivedOnly != 1 || !accounting.valid() {
		t.Fatalf("InsightClaw cost accounting=%+v err=%v", accounting, err)
	}

	metrics := fixture.pipelines.sinks(t, 1).local.snapshot()
	if len(metrics) != 1 || metrics[0].Descriptor().Name != "defenseclaw.agent.reported_cost" {
		t.Fatalf("compat cost metrics = %#v", metrics)
	}
	if metrics[0].CanonicalRecord().Bucket() != observability.BucketAgentLifecycle {
		t.Fatalf("cost compatibility bucket=%q want %q", metrics[0].CanonicalRecord().Bucket(), observability.BucketAgentLifecycle)
	}
	attributes := metrics[0].Attributes()
	if attributes["connector"] != "openclaw" || attributes["gen_ai.provider.name"] != "openai" || attributes["gen_ai.request.model"] != "gpt-4o" {
		t.Fatalf("compat cost labels = %#v", attributes)
	}
	correlation := metrics[0].CanonicalRecord().Correlation()
	if correlation.AgentID != "agent-cost" || correlation.SessionID != "session-cost" {
		t.Fatalf("compat cost correlation = %#v", correlation)
	}
}

func TestOTLPInboundInsightClawUnsupportedMetricRemainsUnsupported(t *testing.T) {
	previousInstance := gatewaylog.SidecarInstanceID()
	gatewaylog.SetSidecarInstanceID("otlp-insightclaw-compat-unsupported-test")
	t.Cleanup(func() { gatewaylog.SetSidecarInstanceID(previousInstance) })

	fixture := newOTLPV8MetricFixture(t)
	api := &APIServer{}
	api.bindOTLPObservabilityRuntime(fixture.runtime)
	now := time.Now().UTC()

	point := &metricspb.NumberDataPoint{
		TimeUnixNano: uint64(now.UnixNano()),
		Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("openclaw.message.channel", "chat"),
		},
		Value: &metricspb.NumberDataPoint_AsInt{AsInt: 2},
	}
	message := &collectormetricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricspb.ResourceMetrics{{
		Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("service.name", "openclaw-gateway"),
		}},
		ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: []*metricspb.Metric{{
			Name: "openclaw.messages.received",
			Unit: "messages",
			Data: &metricspb.Metric_Gauge{Gauge: &metricspb.Gauge{DataPoints: []*metricspb.NumberDataPoint{point}}},
		}}}},
	}}}

	accounting, err := api.importDecodedOTLPRequestV8(
		context.Background(), message, otelSignalMetrics, "openclaw", now,
	)
	if err != nil || accounting.unsupportedIdentity != 1 || !accounting.valid() {
		t.Fatalf("InsightClaw unsupported accounting=%+v err=%v", accounting, err)
	}
	if got := len(fixture.pipelines.sinks(t, 1).local.snapshot()); got != 0 {
		t.Fatalf("unsupported InsightClaw metric emitted %d canonical metrics", got)
	}
}

func TestOTLPInboundInsightClawMemoryReadEventsMapToToolCalls(t *testing.T) {
	previousInstance := gatewaylog.SidecarInstanceID()
	gatewaylog.SetSidecarInstanceID("otlp-insightclaw-compat-memory-read-test")
	t.Cleanup(func() { gatewaylog.SetSidecarInstanceID(previousInstance) })

	fixture := newOTLPV8MetricFixture(t)
	api := &APIServer{}
	api.bindOTLPObservabilityRuntime(fixture.runtime)
	now := time.Now().UTC()

	point := &metricspb.NumberDataPoint{
		TimeUnixNano: uint64(now.UnixNano()),
		Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("openclaw.channel", "terminal"),
			otlpClassifierStringAttribute("openclaw.session.key", "session-memory"),
		},
		Value: &metricspb.NumberDataPoint_AsInt{AsInt: 7},
	}
	message := &collectormetricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricspb.ResourceMetrics{{
		Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("service.name", "openclaw-gateway"),
		}},
		ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: []*metricspb.Metric{{
			Name: "openclaw.memory.read_events",
			Unit: "events",
			Data: &metricspb.Metric_Gauge{Gauge: &metricspb.Gauge{DataPoints: []*metricspb.NumberDataPoint{point}}},
		}}}},
	}}}

	accounting, err := api.importDecodedOTLPRequestV8(
		context.Background(), message, otelSignalMetrics, "openclaw", now,
	)
	if err != nil || accounting.derivedOnly != 1 || !accounting.valid() {
		t.Fatalf("InsightClaw memory-read accounting=%+v err=%v", accounting, err)
	}

	metrics := fixture.pipelines.sinks(t, 1).local.snapshot()
	if len(metrics) != 1 || metrics[0].Descriptor().Name != "defenseclaw.tool.calls" {
		t.Fatalf("compat memory-read metrics = %#v", metrics)
	}
	attributes := metrics[0].Attributes()
	if attributes["gen_ai.tool.name"] != "memory_read" || attributes["tool.provider"] != "terminal" {
		t.Fatalf("compat memory-read labels = %#v", attributes)
	}
}

func TestOTLPInboundInsightClawSessionRepetitionMapsToPresenceScore(t *testing.T) {
	previousInstance := gatewaylog.SidecarInstanceID()
	gatewaylog.SetSidecarInstanceID("otlp-insightclaw-compat-session-score-test")
	t.Cleanup(func() { gatewaylog.SetSidecarInstanceID(previousInstance) })

	fixture := newOTLPV8MetricFixture(t)
	api := &APIServer{}
	api.bindOTLPObservabilityRuntime(fixture.runtime)
	now := time.Now().UTC()

	point := &metricspb.HistogramDataPoint{
		TimeUnixNano: uint64(now.UnixNano()),
		Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("openclaw.session.key", "session-repeat"),
		},
		Count: 1,
		Sum:   0.72,
	}
	message := &collectormetricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricspb.ResourceMetrics{{
		Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("service.name", "openclaw-gateway"),
		}},
		ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: []*metricspb.Metric{{
			Name: "openclaw.session.repetition_score",
			Unit: "score",
			Data: &metricspb.Metric_Histogram{Histogram: &metricspb.Histogram{
				AggregationTemporality: metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				DataPoints:             []*metricspb.HistogramDataPoint{point},
			}},
		}}}},
	}}}

	accounting, err := api.importDecodedOTLPRequestV8(
		context.Background(), message, otelSignalMetrics, "openclaw", now,
	)
	if err != nil || accounting.derivedOnly != 1 || !accounting.valid() {
		t.Fatalf("InsightClaw session-score accounting=%+v err=%v", accounting, err)
	}

	metrics := fixture.pipelines.sinks(t, 1).local.snapshot()
	if len(metrics) != 1 || metrics[0].Descriptor().Name != "defenseclaw.ai.confidence.presence_score" {
		t.Fatalf("compat session-score metrics = %#v", metrics)
	}
	attributes := metrics[0].Attributes()
	if attributes["defenseclaw.metric.name"] != "session_repetition" {
		t.Fatalf("compat session-score labels = %#v", attributes)
	}
}

func TestOTLPInboundInsightClawContextHistorySizeMapsToTokenUsage(t *testing.T) {
	previousInstance := gatewaylog.SidecarInstanceID()
	gatewaylog.SetSidecarInstanceID("otlp-insightclaw-compat-context-size-test")
	t.Cleanup(func() { gatewaylog.SetSidecarInstanceID(previousInstance) })

	fixture := newOTLPV8MetricFixture(t)
	api := &APIServer{}
	api.bindOTLPObservabilityRuntime(fixture.runtime)
	now := time.Now().UTC()

	point := &metricspb.HistogramDataPoint{
		TimeUnixNano: uint64(now.UnixNano()),
		Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("gen_ai.response.model", "gpt-4o-mini"),
			otlpClassifierStringAttribute("openclaw.provider", "openai"),
		},
		Count: 1,
		Sum:   256,
	}
	message := &collectormetricspb.ExportMetricsServiceRequest{ResourceMetrics: []*metricspb.ResourceMetrics{{
		Resource: &resourcepb.Resource{Attributes: []*commonpb.KeyValue{
			otlpClassifierStringAttribute("service.name", "openclaw-gateway"),
		}},
		ScopeMetrics: []*metricspb.ScopeMetrics{{Metrics: []*metricspb.Metric{{
			Name: "openclaw.context.history_user_size",
			Unit: "tokens",
			Data: &metricspb.Metric_Histogram{Histogram: &metricspb.Histogram{
				AggregationTemporality: metricspb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
				DataPoints:             []*metricspb.HistogramDataPoint{point},
			}},
		}}}},
	}}}

	accounting, err := api.importDecodedOTLPRequestV8(
		context.Background(), message, otelSignalMetrics, "openclaw", now,
	)
	if err != nil || accounting.derivedOnly != 1 || !accounting.valid() {
		t.Fatalf("InsightClaw context-size accounting=%+v err=%v", accounting, err)
	}

	metrics := fixture.pipelines.sinks(t, 1).local.snapshot()
	if len(metrics) != 1 || metrics[0].Descriptor().Name != "gen_ai.client.token.usage" {
		t.Fatalf("compat context-size metrics = %#v", metrics)
	}
	attributes := metrics[0].Attributes()
	if attributes["gen_ai.operation.name"] != "context_history_user" || attributes["gen_ai.token.type"] != "input" {
		t.Fatalf("compat context-size labels = %#v", attributes)
	}
}
