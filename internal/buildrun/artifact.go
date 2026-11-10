package buildrun

import "os"

func promote(staging, final string) error {
	if staging == "" || final == "" {
		return nil
	}
	return os.Rename(staging, final)
}

func discard(staging string) {
	if staging == "" {
		return
	}
	_ = os.Remove(staging)
}
