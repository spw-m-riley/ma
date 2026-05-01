package prose

import "testing"

type legacyDisposition struct {
	name    string
	outcome string
	phase   string
	note    string
}

var legacyRuleAudit = []legacyDisposition{
	{name: "goal-of-tool", outcome: "retired", note: "too tool-specific for the final generic rule set"},
	{name: "while-still-keeping-result", outcome: "retired", note: "too phrase-specific to justify a permanent rule"},
	{name: "while-keeping", outcome: "retired", note: "punctuation-heavy rewrite was too aggressive"},
	{name: "humans-and-language-models", outcome: "migrated", phase: "technical-abbreviations"},
	{name: "understandable-to", outcome: "migrated", phase: "wordy-phrases"},
	{name: "you-should-always", outcome: "migrated", phase: "subject-elision"},
	{name: "you-should", outcome: "migrated", phase: "subject-elision"},
	{name: "make-sure-to-keep", outcome: "migrated", phase: "hedging"},
	{name: "make-sure-to-simplify", outcome: "migrated", phase: "hedging"},
	{name: "make-sure-to", outcome: "migrated", phase: "hedging"},
	{name: "remember-to", outcome: "migrated", phase: "hedging"},
	{name: "it-would-be-good-to", outcome: "migrated", phase: "hedging"},
	{name: "in-order-to", outcome: "migrated", phase: "wordy-phrases"},
	{name: "utilize", outcome: "migrated", phase: "wordy-phrases"},
	{name: "please", outcome: "migrated", phase: "hedging"},
	{name: "additionally", outcome: "migrated", phase: "transitions"},
	{name: "where-possible", outcome: "migrated", phase: "hedging"},
	{name: "easy-to-scan", outcome: "migrated", phase: "wordy-phrases"},
	{name: "concise-and-scannable", outcome: "migrated", phase: "wordy-phrases"},
	{name: "unnecessary-filler-words", outcome: "migrated", phase: "wordy-phrases"},
	{name: "to-keep-context-small", outcome: "migrated", phase: "wordy-phrases"},
	{name: "paths-like", outcome: "migrated", phase: "wordy-phrases"},
	{name: "commands-like", outcome: "migrated", phase: "wordy-phrases"},
	{name: "the-prose", outcome: "migrated", phase: "demonstratives"},
	{name: "the-example", outcome: "migrated", phase: "demonstratives"},
	{name: "example-below", outcome: "migrated", phase: "demonstratives"},
	{name: "the-heading", outcome: "migrated", phase: "demonstratives"},
}

func TestRulePhasesExcludeLegacyAndRemainOrdered(t *testing.T) {
	got := make([]string, 0, len(rulePhases()))
	for _, phase := range rulePhases() {
		got = append(got, phase.Name)
	}

	want := []string{
		"contractions",
		"subject-elision",
		"hedging",
		"wordy-phrases",
		"transitions",
		"filler-adverbs",
		"demonstratives",
		"technical-abbreviations",
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d phases, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected phases %v, got %v", want, got)
		}
	}
}

func TestLegacyRuleAuditComplete(t *testing.T) {
	if len(legacyRuleAudit) != 27 {
		t.Fatalf("expected 27 audited legacy rules, got %d", len(legacyRuleAudit))
	}

	seen := make(map[string]struct{}, len(legacyRuleAudit))
	for _, item := range legacyRuleAudit {
		if _, exists := seen[item.name]; exists {
			t.Fatalf("duplicate legacy rule audit entry %q", item.name)
		}
		seen[item.name] = struct{}{}

		switch item.outcome {
		case "migrated":
			if item.phase == "" {
				t.Fatalf("expected migrated rule %q to name its destination phase", item.name)
			}
		case "retired":
			if item.note == "" {
				t.Fatalf("expected retired rule %q to include a retirement note", item.name)
			}
		default:
			t.Fatalf("unexpected outcome %q for legacy rule %q", item.outcome, item.name)
		}
	}
}
