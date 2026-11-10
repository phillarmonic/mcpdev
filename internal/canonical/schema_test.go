package canonical

import (
	"bytes"
	"testing"
)

func TestSchemaRepresentationEqual(t *testing.T) {
	pairs := [][2]string{
		{
			`{"type":"object","required":["b","a"],"properties":{"b":{"type":"string"},"a":{"type":"number"}}}`,
			`{"properties":{"a":{"type":"number"},"b":{"type":"string"}},"required":["a","b"],"type":"object"}`,
		},
		{
			`{"type":["null","string"]}`,
			`{"type":["string","null"]}`,
		},
		{
			`{"type":["string"]}`,
			`{"type":"string"}`,
		},
		{
			`{"enum":[2,1,{"z":1,"a":0}]}`,
			`{"enum":[{"a":0,"z":1},1,2]}`,
		},
		{
			`{"type":"object","required":[]}`,
			`{"type":"object"}`,
		},
		{
			`{"dependentRequired":{"b":["z","a"],"a":["c"]}}`,
			`{"dependentRequired":{"a":["c"],"b":["a","z"]}}`,
		},
	}
	for _, pair := range pairs {
		left, err := Schema([]byte(pair[0]))
		if err != nil {
			t.Fatal(err)
		}
		right, err := Schema([]byte(pair[1]))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(left, right) {
			t.Fatalf("expected equal\n%s\n%s", left, right)
		}
		lHash, err := SchemaDigest([]byte(pair[0]))
		if err != nil {
			t.Fatal(err)
		}
		rHash, err := SchemaDigest([]byte(pair[1]))
		if err != nil {
			t.Fatal(err)
		}
		if lHash != rHash {
			t.Fatalf("digests differ: %s vs %s", lHash, rHash)
		}
	}
}

func TestSchemaKeepsUnknownAndAnnotations(t *testing.T) {
	raw := []byte(`{"x-vendor":{"b":1,"a":2},"description":"keep","default":{"z":1,"a":0},"examples":[{"b":1,"a":0},2],"title":"T"}`)
	got, err := Schema(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"default":{"a":0,"z":1},"description":"keep","examples":[{"a":0,"b":1},2],"title":"T","x-vendor":{"a":2,"b":1}}`)
	if !bytes.Equal(got, want) {
		t.Fatalf("got %s\nwant %s", got, want)
	}
}
