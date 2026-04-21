package mcpctl

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Hello, World!", []string{"hello", "world"}},
		{"ERROR: failed to connect", []string{"error", "failed", "to", "connect"}},
		{"   ", nil},
		{"", nil},
		{"foo-bar_baz", []string{"foobarbaz"}},
	}
	for _, tc := range cases {
		got := Tokenize(tc.in)
		if len(got) == 0 && len(tc.want) == 0 {
			continue
		}
		if len(got) != len(tc.want) {
			t.Fatalf("Tokenize(%q) len = %d, want %d (got %v)", tc.in, len(got), len(tc.want), got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("Tokenize(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestScoreRanksRelevantDocHigher(t *testing.T) {
	docs := [][]string{
		Tokenize("hello world"),
		Tokenize("error: failed to connect to database"),
		Tokenize("info: application started"),
		Tokenize("error handling is important"),
	}
	c := NewCorpus(docs, 0, 0)

	hits := c.TopN(Tokenize("error failed"), 10)
	if len(hits) == 0 {
		t.Fatalf("expected at least one hit")
	}
	if hits[0].DocID != 1 {
		t.Fatalf("expected doc 1 (error: failed to connect) first, got doc %d with hits %+v", hits[0].DocID, hits)
	}
}

func TestScoreEmptyQuery(t *testing.T) {
	docs := [][]string{
		Tokenize("alpha beta"),
		Tokenize("gamma delta"),
	}
	c := NewCorpus(docs, 0, 0)
	scores := c.Score(nil)
	if len(scores) != len(docs) {
		t.Fatalf("Score(nil) len = %d, want %d", len(scores), len(docs))
	}
	for i, s := range scores {
		if s != 0 {
			t.Fatalf("Score(nil)[%d] = %v, want 0", i, s)
		}
	}
}

func TestScoreEmptyCorpus(t *testing.T) {
	c := NewCorpus(nil, 0, 0)
	if got := c.Score(Tokenize("anything")); got != nil {
		t.Fatalf("Score on empty corpus = %v, want nil", got)
	}
	if got := c.TopN(Tokenize("anything"), 5); len(got) != 0 {
		t.Fatalf("TopN on empty corpus = %v, want empty", got)
	}
}

func TestTopNCapsResultsAndTieBreaks(t *testing.T) {
	// Two identical docs produce identical scores; tie-break must be deterministic.
	docs := [][]string{
		Tokenize("error error error"),
		Tokenize("error error error"),
		Tokenize("nothing here"),
	}
	c := NewCorpus(docs, 0, 0)
	hits := c.TopN(Tokenize("error"), 2)
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if hits[0].DocID != 0 || hits[1].DocID != 1 {
		t.Fatalf("tie-break not stable: got %+v", hits)
	}
}

func TestQueryWithNoMatchesReturnsNoHits(t *testing.T) {
	docs := [][]string{
		Tokenize("alpha beta"),
		Tokenize("gamma delta"),
	}
	c := NewCorpus(docs, 0, 0)
	if got := c.TopN(Tokenize("zeta"), 5); len(got) != 0 {
		t.Fatalf("expected no hits, got %v", got)
	}
}
