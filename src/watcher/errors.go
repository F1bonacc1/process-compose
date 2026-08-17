package watcher

import (
	"fmt"
	"sort"
	"strings"
)

// InvalidPatternError reports a glob that doublestar cannot parse. Surfacing it
// matters because an unparseable pattern would otherwise be a watch that
// silently never matches.
type InvalidPatternError struct {
	Root    string
	Pattern string
}

func (e *InvalidPatternError) Error() string {
	return fmt.Sprintf("watch path %q has an invalid pattern %q", e.Root, e.Pattern)
}

// TooManyEntriesError reports a watch that would exceed its entry cap. The cap
// exists because kqueue (macOS, BSD) opens a descriptor per watched entry, and
// exhausting the process limit breaks process launching itself - so failing
// loudly here is far better than the confusing failure that follows.
type TooManyEntriesError struct {
	Process        string
	Root           string
	Scanned        int
	Max            int
	LargestSubdirs []subdirCount
}

type subdirCount struct {
	Path  string
	Count int
}

func (e *TooManyEntriesError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "process %q: watch path %q would add %d directories, exceeding the limit of %d",
		e.Process, e.Root, e.Scanned, e.Max)

	if len(e.LargestSubdirs) > 0 {
		parts := make([]string, 0, len(e.LargestSubdirs))
		for _, sub := range e.LargestSubdirs {
			parts = append(parts, fmt.Sprintf("%s (%d)", sub.Path, sub.Count))
		}
		fmt.Fprintf(&sb, " (largest: %s)", strings.Join(parts, ", "))
	}

	suggestion := "**/node_modules/**"
	if len(e.LargestSubdirs) > 0 {
		suggestion = e.LargestSubdirs[0].Path + "/**"
	}
	fmt.Fprintf(&sb, ". Add an exclude (e.g. exclude: [%q]) or raise watch.max_entries", suggestion)
	return sb.String()
}

// topSubdirs reduces a per-directory tally to the n largest, so the cap error
// can point at what actually blew the budget instead of just reporting a number.
func topSubdirs(counts map[string]int, n int) []subdirCount {
	if len(counts) == 0 {
		return nil
	}
	all := make([]subdirCount, 0, len(counts))
	for dirPath, count := range counts {
		all = append(all, subdirCount{Path: dirPath, Count: count})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Count != all[j].Count {
			return all[i].Count > all[j].Count
		}
		// Deterministic ordering for equal counts keeps the message testable.
		return all[i].Path < all[j].Path
	})
	if len(all) > n {
		all = all[:n]
	}
	return all
}
