// Copyright 2026 Cisco Systems, Inc. and its affiliates
// SPDX-License-Identifier: Apache-2.0

package gateway

import (
	"context"
	"math"
	"strings"

	"github.com/defenseclaw/defenseclaw/internal/observability"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

func (a *APIServer) importInsightClawCompatibilityMetricV8(
	ctx context.Context,
	leaf otlpDecodedLeaf,
	authenticatedSource string,
) (otlpInboundPrimaryDisposition, bool) {
	if a == nil || ctx == nil || leaf.signal != otelSignalMetrics || leaf.metric == nil ||
		!isInsightClawCompatSource(authenticatedSource) {
		return "", false
	}

	name := strings.TrimSpace(leaf.metric.GetName())
	if !strings.HasPrefix(name, "openclaw.") {
		return "", false
	}

	value, ok := insightClawMetricNumericValue(leaf)
	if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
		return otlpInboundInvalidMappedField, true
	}

	// Ignore heartbeat/non-observation points (typically value=0 with openclaw.idle=true).
	if value <= 0 {
		return otlpInboundUnsupportedIdentity, false
	}

	provider := firstInboundMetricLabel(leaf, "gen_ai.provider.name", "openclaw.provider", "openclaw.channel")
	model := firstInboundMetricLabel(leaf, "gen_ai.request.model", "gen_ai.response.model", "model")
	toolName := firstInboundMetricLabel(leaf, "tool.name", "openclaw.tool")
	providerOrChannel := firstInboundMetricLabel(leaf, "openclaw.channel", "openclaw.provider")

	enrichEnvelope := func(envelope observability.FamilyEnvelopeInput) observability.FamilyEnvelopeInput {
		correlation := envelope.Correlation
		if correlation.SessionID == "" {
			if session := firstInboundMetricLabel(leaf, "gen_ai.conversation.id", "openclaw.session.key"); session != "" {
				correlation.SessionID = session
			}
		}
		if correlation.AgentID == "" {
			if agent := firstInboundMetricLabel(leaf, "gen_ai.agent.id"); agent != "" {
				correlation.AgentID = agent
			}
		}
		if correlation.RequestID == "" {
			if requestID := firstInboundMetricLabel(leaf, "defenseclaw.request.id"); requestID != "" {
				correlation.RequestID = requestID
			}
		}
		if correlation.TurnID == "" {
			if turnID := firstInboundMetricLabel(leaf, "defenseclaw.turn.id"); turnID != "" {
				correlation.TurnID = turnID
			}
		}
		envelope.Correlation = correlation
		return envelope
	}

	switch name {
	case "openclaw.memory.read_events":
		count, ok := toNonNegativeInt64(value)
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolCalls),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolCalls(observability.MetricDefenseClawToolCallsInput{
					Envelope:                envelope,
					Value:                   count,
					GenAIToolName:           observability.Present("memory_read"),
					DefenseClawToolProvider: optionalInboundString(providerOrChannel),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.memory.write_events":
		count, ok := toNonNegativeInt64(value)
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolCalls),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolCalls(observability.MetricDefenseClawToolCallsInput{
					Envelope:                envelope,
					Value:                   count,
					GenAIToolName:           observability.Present("memory_write"),
					DefenseClawToolProvider: optionalInboundString(providerOrChannel),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.memory.edit_events":
		count, ok := toNonNegativeInt64(value)
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolCalls),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolCalls(observability.MetricDefenseClawToolCallsInput{
					Envelope:                envelope,
					Value:                   count,
					GenAIToolName:           observability.Present("memory_edit"),
					DefenseClawToolProvider: optionalInboundString(providerOrChannel),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.memory.search_hit":
		count, ok := toNonNegativeInt64(value)
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolCalls),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolCalls(observability.MetricDefenseClawToolCallsInput{
					Envelope:                envelope,
					Value:                   count,
					GenAIToolName:           observability.Present("memory_search_hit"),
					DefenseClawToolProvider: optionalInboundString(providerOrChannel),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.memory.search_miss":
		count, ok := toNonNegativeInt64(value)
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolCalls),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolCalls(observability.MetricDefenseClawToolCallsInput{
					Envelope:                envelope,
					Value:                   count,
					GenAIToolName:           observability.Present("memory_search_miss"),
					DefenseClawToolProvider: optionalInboundString(providerOrChannel),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.context.preparation_duration":
		durationSeconds, ok := scaleDurationToSeconds(value, leaf.metric.GetUnit())
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentGenAIClientOperationDuration),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricGenAIClientOperationDuration(observability.MetricGenAIClientOperationDurationInput{
					Envelope:           envelope,
					Value:              durationSeconds,
					GenAIOperationName: observability.Present("context_preparation"),
					GenAIProviderName:  optionalInboundString(provider),
					GenAIRequestModel:  optionalInboundString(model),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.agent.turn_duration":
		durationSeconds, ok := scaleDurationToSeconds(value, leaf.metric.GetUnit())
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentGenAIClientOperationDuration),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricGenAIClientOperationDuration(observability.MetricGenAIClientOperationDurationInput{
					Envelope:           envelope,
					Value:              durationSeconds,
					GenAIOperationName: observability.Present("agent_turn"),
					GenAIProviderName:  optionalInboundString(provider),
					GenAIRequestModel:  optionalInboundString(model),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.agent.novelty_score":
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawAIConfidenceIdentityScore),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawAIConfidenceIdentityScore(observability.MetricDefenseClawAIConfidenceIdentityScoreInput{
					Envelope:                   envelope,
					Value:                      clampUnitInterval(value),
					DefenseClawMetricEcosystem: observability.Present("openclaw"),
					DefenseClawMetricFramework: observability.Present("insightclaw"),
					DefenseClawMetricName:      observability.Present("agent_novelty"),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.agent.downstream_context_sharing", "openclaw.session.parallelisation_score", "openclaw.session.repetition_score", "openclaw.memory.search_fragmentation":
		metricName := ""
		switch name {
		case "openclaw.agent.downstream_context_sharing":
			metricName = "downstream_context_sharing"
		case "openclaw.session.parallelisation_score":
			metricName = "session_parallelisation"
		case "openclaw.session.repetition_score":
			metricName = "session_repetition"
		case "openclaw.memory.search_fragmentation":
			metricName = "memory_search_fragmentation"
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawAIConfidencePresenceScore),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawAIConfidencePresenceScore(observability.MetricDefenseClawAIConfidencePresenceScoreInput{
					Envelope:                   envelope,
					Value:                      clampUnitInterval(value),
					DefenseClawMetricEcosystem: observability.Present("openclaw"),
					DefenseClawMetricFramework: observability.Present("insightclaw"),
					DefenseClawMetricName:      observability.Present(metricName),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.context.history_memory_size", "openclaw.context.history_other_size", "openclaw.context.history_tool_size", "openclaw.context.history_user_size", "openclaw.context.prompt_size", "openclaw.context.system_size":
		// gen_ai.operation.name is a closed enum with no per-context-slice
		// values (context_history_user, context_prompt, ...), so every
		// context.*_size metric folds onto "chat" -- the sub-slice identity
		// is not representable in this canonical family today.
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentGenAIClientTokenUsage),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricGenAIClientTokenUsage(observability.MetricGenAIClientTokenUsageInput{
					Envelope:           envelope,
					Value:              value,
					GenAIOperationName: observability.Present("chat"),
					GenAIProviderName:  optionalInboundString(provider),
					GenAIRequestModel:  optionalInboundString(model),
					GenAITokenType:     observability.Present("input"),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.cost.usd":
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawAgentReportedCost),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawAgentReportedCost(observability.MetricDefenseClawAgentReportedCostInput{
					Envelope:                   envelope,
					Value:                      value,
					DefenseClawConnectorSource: optionalInboundString(authenticatedSource),
					GenAIProviderName:          optionalInboundString(provider),
					GenAIRequestModel:          optionalInboundString(model),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.llm.tokens.prompt":
		tokenType := "input"
		switch strings.ToLower(strings.TrimSpace(firstInboundMetricLabel(leaf, "token.type"))) {
		case "cache_read":
			tokenType = "cacheRead"
		case "cache_write":
			tokenType = "cacheCreation"
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentGenAIClientTokenUsage),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricGenAIClientTokenUsage(observability.MetricGenAIClientTokenUsageInput{
					Envelope:           envelope,
					Value:              value,
					GenAIOperationName: observability.Present("chat"),
					GenAIProviderName:  optionalInboundString(provider),
					GenAIRequestModel:  optionalInboundString(model),
					GenAITokenType:     observability.Present(tokenType),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.llm.tokens.completion":
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentGenAIClientTokenUsage),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricGenAIClientTokenUsage(observability.MetricGenAIClientTokenUsageInput{
					Envelope:           envelope,
					Value:              value,
					GenAIOperationName: observability.Present("chat"),
					GenAIProviderName:  optionalInboundString(provider),
					GenAIRequestModel:  optionalInboundString(model),
					GenAITokenType:     observability.Present("output"),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.llm.duration":
		durationSeconds, ok := scaleDurationToSeconds(value, leaf.metric.GetUnit())
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentGenAIClientOperationDuration),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricGenAIClientOperationDuration(observability.MetricGenAIClientOperationDurationInput{
					Envelope:           envelope,
					Value:              durationSeconds,
					GenAIOperationName: observability.Present("chat"),
					GenAIProviderName:  optionalInboundString(provider),
					GenAIRequestModel:  optionalInboundString(model),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.tool.calls":
		count, ok := toNonNegativeInt64(value)
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolCalls),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolCalls(observability.MetricDefenseClawToolCallsInput{
					Envelope:                envelope,
					Value:                   count,
					GenAIToolName:           optionalInboundString(toolName),
					DefenseClawToolProvider: optionalInboundString(providerOrChannel),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.tool.duration":
		durationMillis, ok := scaleDurationToMillis(value, leaf.metric.GetUnit())
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolDuration),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolDuration(observability.MetricDefenseClawToolDurationInput{
					Envelope:                envelope,
					Value:                   durationMillis,
					GenAIToolName:           optionalInboundString(toolName),
					DefenseClawToolProvider: optionalInboundString(providerOrChannel),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	case "openclaw.tool.errors":
		count, ok := toNonNegativeInt64(value)
		if !ok {
			return otlpInboundInvalidMappedField, true
		}
		err := a.recordOTLPGeneratedMetricV8(ctx,
			observability.EventName(observability.TelemetryInstrumentDefenseClawToolErrors),
			authenticatedSource,
			func(builder *observability.FamilyBuilder, envelope observability.FamilyEnvelopeInput) (observability.Record, error) {
				envelope = enrichEnvelope(envelope)
				return builder.BuildMetricDefenseClawToolErrors(observability.MetricDefenseClawToolErrorsInput{
					Envelope:      envelope,
					Value:         count,
					GenAIToolName: optionalInboundString(toolName),
				})
			},
		)
		if err != nil {
			return otlpInboundLocalPersistenceFailed, true
		}
		return otlpInboundDerivedOnly, true
	default:
		return "", false
	}
}

func isInsightClawCompatSource(source string) bool {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "openclaw", "insightclaw":
		return true
	default:
		return false
	}
}

func firstInboundMetricLabel(leaf otlpDecodedLeaf, keys ...string) string {
	for _, key := range keys {
		if value, state := leaf.metricPointAttributes.stringValue(key); state == otlpTypedAttributeUnique {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
		if value, state := leaf.resource.attributes.stringValue(key); state == otlpTypedAttributeUnique {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func insightClawMetricNumericValue(leaf otlpDecodedLeaf) (float64, bool) {
	switch {
	case leaf.numberPoint != nil:
		switch value := leaf.numberPoint.Value.(type) {
		case *metricspb.NumberDataPoint_AsDouble:
			return value.AsDouble, true
		case *metricspb.NumberDataPoint_AsInt:
			return float64(value.AsInt), true
		default:
			return 0, false
		}
	case leaf.histogramPoint != nil:
		return leaf.histogramPoint.GetSum(), true
	}
	return 0, false
}

func scaleDurationToSeconds(value float64, unit string) (float64, bool) {
	switch strings.TrimSpace(strings.ToLower(unit)) {
	case "", "s", "second", "seconds":
		return value, true
	case "ms", "millisecond", "milliseconds":
		return value / 1000.0, true
	case "us", "microsecond", "microseconds":
		return value / 1000000.0, true
	case "ns", "nanosecond", "nanoseconds":
		return value / 1000000000.0, true
	default:
		return 0, false
	}
}

func scaleDurationToMillis(value float64, unit string) (float64, bool) {
	switch strings.TrimSpace(strings.ToLower(unit)) {
	case "", "ms", "millisecond", "milliseconds":
		return value, true
	case "s", "second", "seconds":
		return value * 1000.0, true
	case "us", "microsecond", "microseconds":
		return value / 1000.0, true
	case "ns", "nanosecond", "nanoseconds":
		return value / 1000000.0, true
	default:
		return 0, false
	}
}

func toNonNegativeInt64(value float64) (int64, bool) {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	truncated := math.Trunc(value)
	if truncated != value || truncated > math.MaxInt64 {
		return 0, false
	}
	return int64(truncated), true
}

func optionalInboundString(value string) observability.Optional[string] {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return observability.Absent[string]()
	}
	if !observability.IsStableToken(trimmed) {
		return observability.Absent[string]()
	}
	return observability.Present(trimmed)
}

func clampUnitInterval(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
