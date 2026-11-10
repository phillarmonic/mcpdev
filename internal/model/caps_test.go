package model

import "testing"

func TestAdvertisesGatesTemplatesWithResources(t *testing.T) {
	var empty ServerCapabilities
	if empty.Advertises(KindTools) || empty.Advertises(KindTemplates) {
		t.Fatalf("empty: %+v", empty)
	}
	caps := ServerCapabilities{Tools: true, Resources: true}
	if !caps.Advertises(KindTools) || !caps.Advertises(KindTemplates) {
		t.Fatalf("tools+resources: %+v", caps)
	}
	if caps.Advertises(KindPrompts) {
		t.Fatal("prompts must stay unsupported")
	}
	if caps.Advertises(PrimitiveKind("unknown")) {
		t.Fatal("unknown kind")
	}
}
