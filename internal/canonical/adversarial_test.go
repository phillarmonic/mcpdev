package canonical

import (
	"bytes"
	"testing"
)

func TestAdversarialSchemasStayDistinct(t *testing.T) {
	pairs := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "anyOf order",
			a:    `{"anyOf":[{"type":"string"},{"type":"number"}]}`,
			b:    `{"anyOf":[{"type":"number"},{"type":"string"}]}`,
		},
		{
			name: "allOf order",
			a:    `{"allOf":[{"minimum":1},{"maximum":2}]}`,
			b:    `{"allOf":[{"maximum":2},{"minimum":1}]}`,
		},
		{
			name: "oneOf vs anyOf",
			a:    `{"oneOf":[{"type":"string"},{"type":"null"}]}`,
			b:    `{"anyOf":[{"type":"string"},{"type":"null"}]}`,
		},
		{
			name: "prefixItems order",
			a:    `{"prefixItems":[{"type":"string"},{"type":"number"}]}`,
			b:    `{"prefixItems":[{"type":"number"},{"type":"string"}]}`,
		},
		{
			name: "draft-04 tuple items",
			a:    `{"items":[{"type":"string"},{"type":"number"}]}`,
			b:    `{"items":[{"type":"number"},{"type":"string"}]}`,
		},
		{
			name: "additionalProperties false vs missing",
			a:    `{"type":"object","additionalProperties":false}`,
			b:    `{"type":"object"}`,
		},
		{
			name: "additionalProperties true vs missing",
			a:    `{"type":"object","additionalProperties":true}`,
			b:    `{"type":"object"}`,
		},
		{
			name: "additionalProperties empty schema vs true",
			a:    `{"additionalProperties":{}}`,
			b:    `{"additionalProperties":true}`,
		},
		{
			name: "required present vs absent",
			a:    `{"type":"object","required":["id"]}`,
			b:    `{"type":"object"}`,
		},
		{
			name: "enum membership",
			a:    `{"enum":["a","b"]}`,
			b:    `{"enum":["a"]}`,
		},
		{
			name: "nullability",
			a:    `{"type":["string","null"]}`,
			b:    `{"type":"string"}`,
		},
		{
			name: "numeric exclusive vs inclusive",
			a:    `{"type":"number","minimum":1}`,
			b:    `{"type":"number","exclusiveMinimum":1}`,
		},
		{
			name: "$ref vs inline",
			a:    `{"$ref":"#/$defs/t","$defs":{"t":{"type":"string"}}}`,
			b:    `{"type":"string","$defs":{"t":{"type":"string"}}}`,
		},
		{
			name: "description",
			a:    `{"type":"string","description":"a"}`,
			b:    `{"type":"string"}`,
		},
		{
			name: "default",
			a:    `{"type":"string","default":"x"}`,
			b:    `{"type":"string"}`,
		},
		{
			name: "examples",
			a:    `{"type":"string","examples":["a"]}`,
			b:    `{"type":"string"}`,
		},
		{
			name: "extension field",
			a:    `{"type":"string","x-owner":"lab"}`,
			b:    `{"type":"string"}`,
		},
	}
	for _, tc := range pairs {
		left, err := Schema([]byte(tc.a))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		right, err := Schema([]byte(tc.b))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if bytes.Equal(left, right) {
			t.Fatalf("%s collapsed\n%s", tc.name, left)
		}
		lHash, err := SchemaDigest([]byte(tc.a))
		if err != nil {
			t.Fatal(err)
		}
		rHash, err := SchemaDigest([]byte(tc.b))
		if err != nil {
			t.Fatal(err)
		}
		if lHash == rHash {
			t.Fatalf("%s same digest %s", tc.name, lHash)
		}
	}
}

func TestUnknownFieldsRoundTrip(t *testing.T) {
	raw := []byte(`{"type":"object","x-ext":{"keep":true,"order":{"b":2,"a":1}},"additionalProperties":false}`)
	got, err := Schema(raw)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Schema(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, again) {
		t.Fatalf("not stable\n%s\n%s", got, again)
	}
	if !bytes.Contains(got, []byte(`"x-ext"`)) || !bytes.Contains(got, []byte(`"additionalProperties":false`)) {
		t.Fatalf("lost fields: %s", got)
	}
}
