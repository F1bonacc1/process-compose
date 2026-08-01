package types

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/invopop/jsonschema"
	"gopkg.in/yaml.v3"
)

// Namespaces is the set of namespaces a process belongs to.
// It accepts either a single string or a list of strings in YAML and JSON,
// and marshals back to a plain string when it holds a single namespace.
type Namespaces []string

func (n *Namespaces) UnmarshalYAML(node *yaml.Node) error {
	var single string
	if err := node.Decode(&single); err == nil {
		if single == "" {
			*n = nil
		} else {
			*n = Namespaces{single}
		}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err == nil {
		*n = list
		return nil
	}
	return fmt.Errorf("line %d: namespace must be a string or a list of strings", node.Line)
}

func (n Namespaces) MarshalYAML() (any, error) {
	switch len(n) {
	case 0:
		return nil, nil
	case 1:
		return n[0], nil
	default:
		return []string(n), nil
	}
}

func (n *Namespaces) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" {
		*n = nil
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var list []string
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		*n = list
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return fmt.Errorf("namespace must be a string or a list of strings: %w", err)
	}
	if single == "" {
		*n = nil
	} else {
		*n = Namespaces{single}
	}
	return nil
}

func (n Namespaces) MarshalJSON() ([]byte, error) {
	if len(n) == 1 {
		return json.Marshal(n[0])
	}
	return json.Marshal([]string(n))
}

// Normalized drops empty entries, removes duplicates preserving order,
// and falls back to [DefaultNamespace] when nothing remains.
func (n Namespaces) Normalized() Namespaces {
	var normalized Namespaces
	for _, ns := range n {
		if ns == "" || slices.Contains(normalized, ns) {
			continue
		}
		normalized = append(normalized, ns)
	}
	return normalized.OrDefault()
}

// OrDefault returns the namespaces, or [DefaultNamespace] when none are set.
func (n Namespaces) OrDefault() Namespaces {
	if len(n) == 0 {
		return Namespaces{DefaultNamespace}
	}
	return n
}

// Contains reports whether ns is one of the namespaces.
// An empty ns is treated as DefaultNamespace.
func (n Namespaces) Contains(ns string) bool {
	if ns == "" {
		ns = DefaultNamespace
	}
	return slices.Contains(n.OrDefault(), ns)
}

// HasAny reports whether any of the enabled namespaces is one of the namespaces.
func (n Namespaces) HasAny(enabled []string) bool {
	return slices.ContainsFunc(enabled, n.Contains)
}

// Equal reports whether both hold the same namespace set,
// ignoring order and treating empty as [DefaultNamespace].
func (n Namespaces) Equal(other Namespaces) bool {
	a := slices.Clone(n.OrDefault())
	b := slices.Clone(other.OrDefault())
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func (n Namespaces) String() string {
	return strings.Join(n.OrDefault(), ", ")
}

func (Namespaces) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{Type: "string"},
			{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
		},
		Description: "Namespace(s) the process belongs to (string or list of strings, default: \"default\")",
	}
}
