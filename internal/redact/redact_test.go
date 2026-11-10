package redact

import (
	"net/http"
	"strings"
	"testing"
)

func TestSensitiveKeys(t *testing.T) {
	secrets := []string{
		"Authorization",
		"COOKIE",
		"Set-Cookie",
		"X-Api-Key",
		"api_key",
		"access_token",
		"client_secret",
		"password",
		"MCPDEV_TOKEN",
	}
	for _, k := range secrets {
		if !Sensitive(k, []string{"mcpdev_token"}) {
			t.Fatalf("expected sensitive %q", k)
		}
	}
	safe := []string{"tokenizer", "authorization_endpoint", "content-type", "protocol", "public_key"}
	for _, k := range safe {
		if Sensitive(k, nil) {
			t.Fatalf("false positive %q", k)
		}
	}
}

func TestHeaderMixedCase(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer super-secret")
	h.Set("Content-Type", "application/json")
	h.Add("X-Api-Key", "abc")
	out := Header(h, Options{})
	if out.Get("Authorization") != DefaultReplacement {
		t.Fatalf("authorization: %q", out.Get("Authorization"))
	}
	if out.Get("X-Api-Key") != DefaultReplacement {
		t.Fatalf("api key: %q", out.Get("X-Api-Key"))
	}
	if out.Get("Content-Type") != "application/json" {
		t.Fatalf("content-type mutated: %q", out.Get("Content-Type"))
	}
	if h.Get("Authorization") != "Bearer super-secret" {
		t.Fatal("input header mutated")
	}
}

func TestNestedMapsAndMetadata(t *testing.T) {
	in := map[string]any{
		"headers": map[string]any{
			"Authorization": "Bearer nest",
			"Accept":        "application/json",
		},
		"_meta": map[string]any{
			"token": "meta-secret",
			"era":   "2026-07-28",
		},
		"config": map[string]string{
			"PASSWORD": "hunter2",
			"timeout":  "10s",
		},
		"algorithm": "token-bucket",
	}
	out, ok := Any(in, Options{}).(map[string]any)
	if !ok {
		t.Fatal("type")
	}
	headers := out["headers"].(map[string]any)
	if headers["Authorization"] != DefaultReplacement {
		t.Fatalf("nested header %v", headers["Authorization"])
	}
	meta := out["_meta"].(map[string]any)
	if meta["token"] != DefaultReplacement {
		t.Fatalf("meta token %v", meta["token"])
	}
	if meta["era"] != "2026-07-28" {
		t.Fatal("era")
	}
	cfg := out["config"].(map[string]string)
	if cfg["PASSWORD"] != DefaultReplacement || cfg["timeout"] != "10s" {
		t.Fatalf("config %#v", cfg)
	}
	if out["algorithm"] != "token-bucket" {
		t.Fatalf("value false positive: %v", out["algorithm"])
	}
}

func TestURLCredentials(t *testing.T) {
	raw := "https://user:hunter2@example.com/path?token=abc&q=ok"
	got := URLString(raw, Options{})
	if strings.Contains(got, "hunter2") || strings.Contains(got, "abc") {
		t.Fatalf("leaked: %s", got)
	}
	if !strings.Contains(got, "q=ok") {
		t.Fatalf("lost query: %s", got)
	}
	if URLString("not a url token=abc", Options{}) != "not a url token=abc" {
		t.Fatal("non-URL scanned")
	}
}

func TestTextHeaderLines(t *testing.T) {
	got := Text("Authorization: Bearer abc\nready", Options{})
	if strings.Contains(got, "abc") {
		t.Fatalf("leaked: %q", got)
	}
	if !strings.Contains(got, "ready") {
		t.Fatalf("lost line: %q", got)
	}
}

func TestArgvAndExtraEnv(t *testing.T) {
	args := []string{
		"mcpdev",
		"--header", "Authorization: Bearer xyz",
		"--env", "GITHUB_PAT=ghp_secret",
		"API_KEY=k",
		"https://u:p@127.0.0.1:1/mcp",
	}
	got := Argv(args, Options{ExtraKeys: []string{"GITHUB_PAT"}})
	joined := strings.Join(got, " ")
	if strings.Contains(joined, "xyz") || strings.Contains(joined, "ghp_secret") || strings.Contains(joined, "API_KEY=k") {
		t.Fatalf("leaked argv: %q", got)
	}
	if !strings.Contains(got[2], DefaultReplacement) {
		t.Fatalf("header arg: %q", got[2])
	}
}
