package insightclaw

import (
	"strings"
	"unicode"
)

// ComputeNoveltyScore computes a novelty ratio (0.0–1.0) by comparing the
// output text against the parent context. It mirrors the InsightClaw plugin's
// getNoveltyScore: tokenizes both texts, then returns the fraction of output
// tokens that do NOT appear in the parent context.
//
// A score of 0.0 means all output tokens are present in the parent (fully
// redundant). A score of 1.0 means no output tokens appear in the parent (fully
// novel).
func ComputeNoveltyScore(output, parentContext string) float64 {
	outputTokens := tokenize(output)
	if len(outputTokens) == 0 {
		return 0.0
	}

	parentSet := tokenSet(tokenize(parentContext))

	novel := 0
	for _, tok := range outputTokens {
		if _, found := parentSet[tok]; !found {
			novel++
		}
	}
	return float64(novel) / float64(len(outputTokens))
}

// tokenize splits text into lowercase word tokens.
func tokenize(text string) []string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		t := strings.ToLower(w)
		if len(t) > 1 { // skip single-char tokens
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func tokenSet(tokens []string) map[string]struct{} {
	s := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		s[t] = struct{}{}
	}
	return s
}
