package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNamespaces_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    Namespaces
		wantErr bool
	}{
		{name: "scalar", yaml: `namespace: foo`, want: Namespaces{"foo"}},
		{name: "sequence", yaml: `namespace: [foo, bar]`, want: Namespaces{"foo", "bar"}},
		{name: "block sequence", yaml: "namespace:\n  - foo\n  - bar", want: Namespaces{"foo", "bar"}},
		{name: "null", yaml: `namespace: null`, want: nil},
		{name: "empty string", yaml: `namespace: ""`, want: nil},
		{name: "empty sequence", yaml: `namespace: []`, want: Namespaces{}},
		{name: "mapping is invalid", yaml: `namespace: {foo: bar}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var holder struct {
				Namespace Namespaces `yaml:"namespace"`
			}
			err := yaml.Unmarshal([]byte(tt.yaml), &holder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(holder.Namespace, tt.want) {
				t.Errorf("Unmarshal = %#v, want %#v", holder.Namespace, tt.want)
			}
		})
	}
}

func TestNamespaces_MarshalYAML(t *testing.T) {
	tests := []struct {
		name string
		ns   Namespaces
		want string
	}{
		{name: "single as scalar", ns: Namespaces{"foo"}, want: "namespace: foo\n"},
		{name: "multiple as sequence", ns: Namespaces{"foo", "bar"}, want: "namespace:\n    - foo\n    - bar\n"},
		{name: "empty omitted", ns: nil, want: "{}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holder := struct {
				Namespace Namespaces `yaml:"namespace,omitempty"`
			}{Namespace: tt.ns}
			data, err := yaml.Marshal(holder)
			if err != nil {
				t.Fatalf("Marshal error = %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("Marshal = %q, want %q", string(data), tt.want)
			}
		})
	}
}

func TestNamespaces_YAMLRoundTrip(t *testing.T) {
	for _, ns := range []Namespaces{{"foo"}, {"foo", "bar"}} {
		holder := struct {
			Namespace Namespaces `yaml:"namespace,omitempty"`
		}{Namespace: ns}
		data, err := yaml.Marshal(holder)
		if err != nil {
			t.Fatalf("Marshal error = %v", err)
		}
		holder.Namespace = nil
		if err := yaml.Unmarshal(data, &holder); err != nil {
			t.Fatalf("Unmarshal error = %v", err)
		}
		if !reflect.DeepEqual(holder.Namespace, ns) {
			t.Errorf("round trip = %#v, want %#v", holder.Namespace, ns)
		}
	}
}

func TestNamespaces_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    Namespaces
		wantErr bool
	}{
		{name: "string", json: `{"namespace": "foo"}`, want: Namespaces{"foo"}},
		{name: "array", json: `{"namespace": ["foo", "bar"]}`, want: Namespaces{"foo", "bar"}},
		{name: "null", json: `{"namespace": null}`, want: nil},
		{name: "absent", json: `{}`, want: nil},
		{name: "empty string", json: `{"namespace": ""}`, want: nil},
		{name: "number is invalid", json: `{"namespace": 42}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var holder struct {
				Namespace Namespaces `json:"namespace,omitempty"`
			}
			err := json.Unmarshal([]byte(tt.json), &holder)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(holder.Namespace, tt.want) {
				t.Errorf("Unmarshal = %#v, want %#v", holder.Namespace, tt.want)
			}
		})
	}
}

func TestNamespaces_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		ns   Namespaces
		want string
	}{
		{name: "single as string", ns: Namespaces{"foo"}, want: `{"namespace":"foo"}`},
		{name: "multiple as array", ns: Namespaces{"foo", "bar"}, want: `{"namespace":["foo","bar"]}`},
		{name: "empty omitted", ns: nil, want: `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holder := struct {
				Namespace Namespaces `json:"namespace,omitempty"`
			}{Namespace: tt.ns}
			data, err := json.Marshal(holder)
			if err != nil {
				t.Fatalf("Marshal error = %v", err)
			}
			if string(data) != tt.want {
				t.Errorf("Marshal = %s, want %s", string(data), tt.want)
			}
		})
	}
}

func TestNamespaces_Equal(t *testing.T) {
	tests := []struct {
		name string
		a, b Namespaces
		want bool
	}{
		{name: "same", a: Namespaces{"a"}, b: Namespaces{"a"}, want: true},
		{name: "different", a: Namespaces{"a"}, b: Namespaces{"b"}, want: false},
		{name: "order insensitive", a: Namespaces{"a", "b"}, b: Namespaces{"b", "a"}, want: true},
		{name: "subset", a: Namespaces{"a", "b"}, b: Namespaces{"a"}, want: false},
		{name: "nil equals default", a: nil, b: Namespaces{DefaultNamespace}, want: true},
		{name: "both nil", a: nil, b: nil, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Equal(tt.b); got != tt.want {
				t.Errorf("Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNamespaces_Helpers(t *testing.T) {
	if got := (Namespaces{}).OrDefault(); !reflect.DeepEqual(got, Namespaces{DefaultNamespace}) {
		t.Errorf("OrDefault() = %#v", got)
	}
	if got := (Namespaces{"a"}).OrDefault(); !reflect.DeepEqual(got, Namespaces{"a"}) {
		t.Errorf("OrDefault() = %#v", got)
	}
	if !(Namespaces{"a", "b"}).Contains("b") {
		t.Error("Contains should match a listed namespace")
	}
	if (Namespaces{"a"}).Contains("b") {
		t.Error("Contains should not match an unlisted namespace")
	}
	if !(Namespaces(nil)).Contains("") {
		t.Error("empty query should match the default namespace of an empty set")
	}
	if !(Namespaces{"a", "b"}).HasAny([]string{"c", "b"}) {
		t.Error("HasAny should match on intersection")
	}
	if (Namespaces{"a"}).HasAny([]string{"b", "c"}) {
		t.Error("HasAny should not match without intersection")
	}
	if got := (Namespaces{"b", "", "a", "b"}).Normalized(); !reflect.DeepEqual(got, Namespaces{"b", "a"}) {
		t.Errorf("Normalized() = %#v, want [b a]", got)
	}
	if got := (Namespaces{"", ""}).Normalized(); !reflect.DeepEqual(got, Namespaces{DefaultNamespace}) {
		t.Errorf("Normalized() = %#v, want [default]", got)
	}
	if got := (Namespaces{"a", "b"}).String(); got != "a, b" {
		t.Errorf("String() = %q", got)
	}
	if got := (Namespaces(nil)).String(); got != DefaultNamespace {
		t.Errorf("String() = %q", got)
	}
}
