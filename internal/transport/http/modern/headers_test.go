package modern

import "testing"

func TestEncodeHeaderValue(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"us-west1", "us-west1"},
		{"Hello, 世界", "=?base64?SGVsbG8sIOS4lueVjA==?="},
		{" padded ", "=?base64?IHBhZGRlZCA=?="},
		{"line1\nline2", "=?base64?bGluZTEKbGluZTI=?="},
		{"=?base64?literal?=", "=?base64?PT9iYXNlNjQ/bGl0ZXJhbD89?="},
		{"echo", "echo"},
		{"file:///projects/myapp/config.json", "file:///projects/myapp/config.json"},
	}
	for _, tc := range tests {
		if got := EncodeHeaderValue(tc.in); got != tc.want {
			t.Errorf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}
