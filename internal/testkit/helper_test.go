package testkit

import "testing"

func closeLab(t testing.TB, lab interface{ Close() error }) {
	t.Helper()
	t.Cleanup(func() {
		if err := lab.Close(); err != nil {
			t.Errorf("close fixture: %v", err)
		}
	})
}
