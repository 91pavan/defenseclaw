// Copyright 2026 Cisco Systems, Inc. and its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package insightclaw

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func setupTestAdapter(t *testing.T) (*Adapter, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	adapter, err := NewAdapter(meter, Config{Enabled: true})
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}
	return adapter, reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return rm
}

func findMetric(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

func TestNilAdapterIsNoop(t *testing.T) {
	var a *Adapter
	// These should not panic.
	a.EmitMessageReceived(context.Background(), "test")
	a.EmitMessageSent(context.Background(), "test")
	a.EmitToolCall(context.Background(), "bash", "session-1")
	a.EmitToolError(context.Background(), "bash")
	a.EmitTokenUsage(context.Background(), "claudecode", "claude-4", 100, 50, 150)
	a.EmitLLMDuration(context.Background(), "claude-4", 123.4)
	a.EmitAgentTurnDuration(context.Background(), "claude-4", "agent-1", 500.0)
	a.EmitWebhookReceived(context.Background(), "slack", "message")
	a.EmitSessionState(context.Background(), "active", "")
}

func TestDisabledConfigReturnsNil(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	adapter, err := NewAdapter(meter, Config{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter != nil {
		t.Fatal("expected nil adapter when disabled")
	}
}

func TestEmitMessageReceived(t *testing.T) {
	adapter, reader := setupTestAdapter(t)
	ctx := context.Background()

	adapter.EmitMessageReceived(ctx, "claudecode")
	adapter.EmitMessageReceived(ctx, "claudecode")
	adapter.EmitMessageReceived(ctx, "codex")

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "openclaw.messages.received")
	if m == nil {
		t.Fatal("metric openclaw.messages.received not found")
	}

	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("expected Sum[int64], got %T", m.Data)
	}

	// Should have 2 data points (claudecode=2, codex=1).
	if len(sum.DataPoints) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(sum.DataPoints))
	}

	total := int64(0)
	for _, dp := range sum.DataPoints {
		total += dp.Value
	}
	if total != 3 {
		t.Fatalf("expected total=3, got %d", total)
	}
}

func TestEmitTokenUsage(t *testing.T) {
	adapter, reader := setupTestAdapter(t)
	ctx := context.Background()

	adapter.EmitTokenUsage(ctx, "claudecode", "claude-4", 100, 50, 150)

	rm := collectMetrics(t, reader)

	for _, name := range []string{
		"openclaw.llm.tokens.prompt",
		"openclaw.llm.tokens.completion",
		"openclaw.llm.tokens.total",
		"openclaw.llm.requests",
	} {
		m := findMetric(rm, name)
		if m == nil {
			t.Errorf("metric %s not found", name)
		}
	}

	// Verify prompt tokens value.
	m := findMetric(rm, "openclaw.llm.tokens.prompt")
	sum := m.Data.(metricdata.Sum[int64])
	if len(sum.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(sum.DataPoints))
	}
	if sum.DataPoints[0].Value != 100 {
		t.Fatalf("expected prompt tokens=100, got %d", sum.DataPoints[0].Value)
	}

	// Verify model attribute.
	found := false
	for _, attr := range sum.DataPoints[0].Attributes.ToSlice() {
		if attr.Key == attribute.Key("gen_ai.request.model") && attr.Value.AsString() == "claude-4" {
			found = true
		}
	}
	if !found {
		t.Error("gen_ai.request.model attribute not found on token metric")
	}
}

func TestEmitToolCall(t *testing.T) {
	adapter, reader := setupTestAdapter(t)
	ctx := context.Background()

	adapter.EmitToolCall(ctx, "bash", "session-abc")

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "openclaw.tool.calls")
	if m == nil {
		t.Fatal("metric openclaw.tool.calls not found")
	}

	sum := m.Data.(metricdata.Sum[int64])
	if len(sum.DataPoints) != 1 || sum.DataPoints[0].Value != 1 {
		t.Fatalf("expected 1 tool call, got %v", sum.DataPoints)
	}

	// Verify session.key attribute is present.
	found := false
	for _, attr := range sum.DataPoints[0].Attributes.ToSlice() {
		if attr.Key == attribute.Key("session.key") && attr.Value.AsString() == "session-abc" {
			found = true
		}
	}
	if !found {
		t.Error("session.key attribute not found on tool.calls metric")
	}
}

func TestCustomPrefix(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	adapter, err := NewAdapter(meter, Config{Enabled: true, Prefix: "myprefix"})
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}

	adapter.EmitMessageReceived(context.Background(), "test")

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "myprefix.messages.received")
	if m == nil {
		t.Fatal("metric myprefix.messages.received not found (custom prefix)")
	}
}

func TestEmitToolLoop(t *testing.T) {
	adapter, reader := setupTestAdapter(t)
	ctx := context.Background()

	adapter.EmitToolLoop(ctx, "bash", "consecutive_args", "alert", "LOW")
	adapter.EmitToolLoop(ctx, "bash", "consecutive_args", "alert", "HIGH")

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "openclaw.tool.loop")
	if m == nil {
		t.Fatal("metric openclaw.tool.loop not found")
	}

	sum := m.Data.(metricdata.Sum[int64])
	total := int64(0)
	for _, dp := range sum.DataPoints {
		total += dp.Value
	}
	if total != 2 {
		t.Fatalf("expected total=2 tool loop events, got %d", total)
	}

	// Verify attributes on first data point.
	foundTool := false
	for _, dp := range sum.DataPoints {
		for _, attr := range dp.Attributes.ToSlice() {
			if attr.Key == attribute.Key("gen_ai.tool.name") && attr.Value.AsString() == "bash" {
				foundTool = true
			}
		}
	}
	if !foundTool {
		t.Error("gen_ai.tool.name attribute not found on tool.loop metric")
	}
}

func TestEmitRunAttempt(t *testing.T) {
	adapter, reader := setupTestAdapter(t)
	ctx := context.Background()

	adapter.EmitRunAttempt(ctx, 1)
	adapter.EmitRunAttempt(ctx, 2)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "openclaw.run.attempt")
	if m == nil {
		t.Fatal("metric openclaw.run.attempt not found")
	}

	sum := m.Data.(metricdata.Sum[int64])
	total := int64(0)
	for _, dp := range sum.DataPoints {
		total += dp.Value
	}
	if total != 2 {
		t.Fatalf("expected total=2 run attempts, got %d", total)
	}
}

func TestEmitQueueWait(t *testing.T) {
	adapter, reader := setupTestAdapter(t)
	ctx := context.Background()

	adapter.EmitQueueWait(ctx, "judge_persist", 42.5)
	adapter.EmitQueueWait(ctx, "judge_persist", 100.0)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "openclaw.queue.wait_ms")
	if m == nil {
		t.Fatal("metric openclaw.queue.wait_ms not found")
	}

	hist := m.Data.(metricdata.Histogram[float64])
	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected 1 data point (same lane), got %d", len(hist.DataPoints))
	}
	if hist.DataPoints[0].Count != 2 {
		t.Fatalf("expected 2 observations, got %d", hist.DataPoints[0].Count)
	}

	// Verify lane attribute.
	found := false
	for _, attr := range hist.DataPoints[0].Attributes.ToSlice() {
		if attr.Key == attribute.Key("lane") && attr.Value.AsString() == "judge_persist" {
			found = true
		}
	}
	if !found {
		t.Error("lane attribute not found on queue.wait_ms metric")
	}
}

func TestEmitSessionStuck(t *testing.T) {
	adapter, reader := setupTestAdapter(t)
	ctx := context.Background()

	adapter.EmitSessionStuck(ctx, "waiting_tool", 5000.0)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "openclaw.session.stuck")
	if m == nil {
		t.Fatal("metric openclaw.session.stuck not found")
	}
	sum := m.Data.(metricdata.Sum[int64])
	if len(sum.DataPoints) != 1 || sum.DataPoints[0].Value != 1 {
		t.Fatalf("expected 1 stuck event, got %v", sum.DataPoints)
	}

	m2 := findMetric(rm, "openclaw.session.stuck_age_ms")
	if m2 == nil {
		t.Fatal("metric openclaw.session.stuck_age_ms not found")
	}
	hist := m2.Data.(metricdata.Histogram[float64])
	if len(hist.DataPoints) != 1 || hist.DataPoints[0].Count != 1 {
		t.Fatalf("expected 1 stuck_age observation, got %v", hist.DataPoints)
	}
}

// --- Phase 4: Experimental metric tests ---

func setupExperimentalAdapter(t *testing.T) (*Adapter, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	adapter, err := NewAdapter(meter, Config{Enabled: true, Experimental: true})
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}
	return adapter, reader
}

func TestExperimentalDisabled_NilInstruments(t *testing.T) {
	adapter, _ := setupTestAdapter(t) // non-experimental
	ctx := context.Background()

	// Should not panic — nil guard in EmitContextWindow.
	adapter.EmitContextWindow(ctx, "claude-4", 200000, 150000)
	adapter.EmitMemoryRead(ctx, "memory_read", "sess-1")
	adapter.EmitMemoryWrite(ctx, "memory_write", "sess-1")
	adapter.EmitSessionParallelisationScore(ctx, "sess-1", 0.5)
	adapter.EmitSessionRepetitionScore(ctx, "sess-1", 0.2)

	if adapter.Experimental() {
		t.Fatal("expected Experimental()=false for non-experimental adapter")
	}
}

func TestEmitContextWindow(t *testing.T) {
	adapter, reader := setupExperimentalAdapter(t)
	ctx := context.Background()

	adapter.EmitContextWindow(ctx, "claude-4", 200000, 150000)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "openclaw.context.limit")
	if m == nil {
		t.Fatal("metric openclaw.context.limit not found")
	}

	m2 := findMetric(rm, "openclaw.context.used")
	if m2 == nil {
		t.Fatal("metric openclaw.context.used not found")
	}

	m3 := findMetric(rm, "openclaw.context.utilization")
	if m3 == nil {
		t.Fatal("metric openclaw.context.utilization not found")
	}

	hist := m3.Data.(metricdata.Histogram[float64])
	if len(hist.DataPoints) != 1 || hist.DataPoints[0].Count != 1 {
		t.Fatalf("expected 1 utilization observation, got %v", hist.DataPoints)
	}
}

func TestEmitMemoryReadWrite(t *testing.T) {
	adapter, reader := setupExperimentalAdapter(t)
	ctx := context.Background()

	adapter.EmitMemoryRead(ctx, "memory_search", "sess-1")
	adapter.EmitMemoryRead(ctx, "memory_search", "sess-1")
	adapter.EmitMemoryWrite(ctx, "memory_write", "sess-1")

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "openclaw.memory.read")
	if m == nil {
		t.Fatal("metric openclaw.memory.read not found")
	}
	sum := m.Data.(metricdata.Sum[int64])
	total := int64(0)
	for _, dp := range sum.DataPoints {
		total += dp.Value
	}
	if total != 2 {
		t.Fatalf("expected 2 memory reads, got %d", total)
	}

	m2 := findMetric(rm, "openclaw.memory.write")
	if m2 == nil {
		t.Fatal("metric openclaw.memory.write not found")
	}
	sum2 := m2.Data.(metricdata.Sum[int64])
	if len(sum2.DataPoints) != 1 || sum2.DataPoints[0].Value != 1 {
		t.Fatalf("expected 1 memory write, got %v", sum2.DataPoints)
	}
}

func TestEmitSessionScores(t *testing.T) {
	adapter, reader := setupExperimentalAdapter(t)
	ctx := context.Background()

	adapter.EmitSessionParallelisationScore(ctx, "sess-1", 0.75)
	adapter.EmitSessionRepetitionScore(ctx, "sess-1", 0.3)

	rm := collectMetrics(t, reader)

	m := findMetric(rm, "openclaw.session.parallelisation_score")
	if m == nil {
		t.Fatal("metric openclaw.session.parallelisation_score not found")
	}
	hist := m.Data.(metricdata.Histogram[float64])
	if len(hist.DataPoints) != 1 || hist.DataPoints[0].Count != 1 {
		t.Fatalf("expected 1 parallelisation observation, got %v", hist.DataPoints)
	}

	m2 := findMetric(rm, "openclaw.session.repetition_score")
	if m2 == nil {
		t.Fatal("metric openclaw.session.repetition_score not found")
	}
	hist2 := m2.Data.(metricdata.Histogram[float64])
	if len(hist2.DataPoints) != 1 || hist2.DataPoints[0].Count != 1 {
		t.Fatalf("expected 1 repetition observation, got %v", hist2.DataPoints)
	}
}

func TestExperimentalFlag(t *testing.T) {
	adapter, _ := setupExperimentalAdapter(t)
	if !adapter.Experimental() {
		t.Fatal("expected Experimental()=true")
	}
}

func TestEmitContextComposition(t *testing.T) {
adapter, reader := setupExperimentalAdapter(t)
ctx := context.Background()

comp := ContextComposition{
SystemBytes:      512,
HistoryToolBytes: 1024,
HistoryUserBytes: 2048,
HistoryMemBytes:  256,
HistoryOther:     128,
PromptBytes:      768,
}
adapter.EmitContextComposition(ctx, "agent-1", comp)

var rm metricdata.ResourceMetrics
if err := reader.Collect(ctx, &rm); err != nil {
t.Fatal(err)
}

for _, name := range []string{
"openclaw.context.system_size",
"openclaw.context.history_tool_size",
"openclaw.context.history_user_size",
"openclaw.context.history_memory_size",
"openclaw.context.history_other_size",
"openclaw.context.prompt_size",
} {
m := findMetric(rm, name)
if m == nil {
t.Errorf("metric %s not found", name)
continue
}
hist := m.Data.(metricdata.Histogram[float64])
if len(hist.DataPoints) != 1 || hist.DataPoints[0].Count != 1 {
t.Errorf("metric %s: expected 1 observation, got %v", name, hist.DataPoints)
}
}
}

func TestEmitNoveltyScore(t *testing.T) {
adapter, reader := setupExperimentalAdapter(t)
ctx := context.Background()

adapter.EmitNoveltyScore(ctx, "sub-agent-1", 0.75)

var rm metricdata.ResourceMetrics
if err := reader.Collect(ctx, &rm); err != nil {
t.Fatal(err)
}

m := findMetric(rm, "openclaw.agent.novelty_score")
if m == nil {
t.Fatal("metric openclaw.agent.novelty_score not found")
}
hist := m.Data.(metricdata.Histogram[float64])
if len(hist.DataPoints) != 1 || hist.DataPoints[0].Count != 1 {
t.Fatalf("expected 1 observation, got %v", hist.DataPoints)
}
// Verify agent ID attribute
attrs := hist.DataPoints[0].Attributes
val, ok := attrs.Value(attribute.Key("gen_ai.agent.id"))
if !ok || val.AsString() != "sub-agent-1" {
t.Fatalf("expected gen_ai.agent.id=sub-agent-1, got %v", attrs)
}
}

func TestComputeNoveltyScore(t *testing.T) {
tests := []struct {
name     string
output   string
parent   string
wantLow  float64
wantHigh float64
}{
{"empty output", "", "anything here", 0.0, 0.0},
{"fully novel", "completely brand new unique content here", "the fox jumped over the lazy dog", 0.9, 1.0},
{"fully redundant", "the quick fox jumped", "the quick brown fox jumped over the lazy dog", 0.0, 0.1},
{"partial", "the fox discovered a new planet orbiting mars", "the fox jumped over the lazy dog in the park", 0.3, 0.8},
}
for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
score := ComputeNoveltyScore(tt.output, tt.parent)
if score < tt.wantLow || score > tt.wantHigh {
t.Errorf("ComputeNoveltyScore() = %v, want [%v, %v]", score, tt.wantLow, tt.wantHigh)
}
})
}
}

func TestContextCompositionNilSafe(t *testing.T) {
// Should not panic on nil adapter
var adapter *Adapter
adapter.EmitContextComposition(context.Background(), "x", ContextComposition{})
adapter.EmitNoveltyScore(context.Background(), "x", 0.5)
}
