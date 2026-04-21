package mcpctl

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Default Okapi BM25 parameters. These match the `okapibm25` TS reference used
// by the prior prototype in repl-it-web PR #74337.
const (
	DefaultK1 = 1.5
	DefaultB  = 0.75
)

// Hit is a single scored document result from a BM25 search.
type Hit struct {
	DocID int
	Score float64
}

// Corpus holds pre-tokenized documents and the statistics needed to score
// queries using Okapi BM25.
type Corpus struct {
	docs     [][]string       // tokenized docs (one slice per doc)
	docLens  []int            // doc length per doc
	avgDL    float64          // average doc length across the corpus
	docFreq  map[string]int   // term -> number of docs containing term
	idfCache map[string]float64
	termFreq []map[string]int // per-doc term frequency
	k1       float64
	b        float64
}

// NewCorpus builds a scoring corpus over the provided tokenized documents.
// Pass k1 and b as 0 to use the Okapi defaults (1.5 and 0.75 respectively).
func NewCorpus(docs [][]string, k1, b float64) *Corpus {
	if k1 <= 0 {
		k1 = DefaultK1
	}
	if b <= 0 {
		b = DefaultB
	}

	c := &Corpus{
		docs:     docs,
		docLens:  make([]int, len(docs)),
		docFreq:  make(map[string]int),
		idfCache: make(map[string]float64),
		termFreq: make([]map[string]int, len(docs)),
		k1:       k1,
		b:        b,
	}

	var totalLen int
	for i, tokens := range docs {
		c.docLens[i] = len(tokens)
		totalLen += len(tokens)

		tf := make(map[string]int, len(tokens))
		for _, t := range tokens {
			tf[t]++
		}
		c.termFreq[i] = tf

		for term := range tf {
			c.docFreq[term]++
		}
	}
	if len(docs) > 0 {
		c.avgDL = float64(totalLen) / float64(len(docs))
	}
	return c
}

// idf returns the inverse document frequency for a term.
// Uses the classic Okapi form: log(1 + (N - n + 0.5) / (n + 0.5)).
func (c *Corpus) idf(term string) float64 {
	if v, ok := c.idfCache[term]; ok {
		return v
	}
	n := c.docFreq[term]
	N := len(c.docs)
	idf := math.Log(1 + (float64(N-n)+0.5)/(float64(n)+0.5))
	c.idfCache[term] = idf
	return idf
}

// Score returns a BM25 score for the query against every document in the corpus.
// Result[i] is the score for doc i. An empty corpus returns nil; an empty query
// returns a zero slice.
func (c *Corpus) Score(query []string) []float64 {
	if len(c.docs) == 0 {
		return nil
	}
	scores := make([]float64, len(c.docs))
	if len(query) == 0 {
		return scores
	}
	for _, term := range query {
		idf := c.idf(term)
		if idf == 0 {
			continue
		}
		for i, tf := range c.termFreq {
			f := float64(tf[term])
			if f == 0 {
				continue
			}
			dl := float64(c.docLens[i])
			num := f * (c.k1 + 1)
			den := f + c.k1*(1-c.b+c.b*dl/c.avgDL)
			scores[i] += idf * num / den
		}
	}
	return scores
}

// TopN returns the top-n highest-scoring documents for the query, sorted by
// score descending. Documents with a zero or negative score are omitted.
// Ties break by lower DocID for deterministic ordering.
func (c *Corpus) TopN(query []string, n int) []Hit {
	scores := c.Score(query)
	hits := make([]Hit, 0, len(scores))
	for i, s := range scores {
		if s > 0 {
			hits = append(hits, Hit{DocID: i, Score: s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].DocID < hits[j].DocID
	})
	if n > 0 && len(hits) > n {
		hits = hits[:n]
	}
	return hits
}

// Tokenize lowercases the string, splits on whitespace, and strips non-alphanumeric
// characters. Empty tokens are discarded. Sufficient for log-line search.
func Tokenize(s string) []string {
	fields := strings.Fields(strings.ToLower(s))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		clean := stripNonAlnum(f)
		if clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func stripNonAlnum(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
