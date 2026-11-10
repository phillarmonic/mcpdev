package modern

import "slices"

// Select returns the first client-preferred version that the server also lists.
// An empty string means there is no mutually supported modern version.
func Select(clientPreferred, serverSupported []string) string {
	if len(clientPreferred) == 0 {
		clientPreferred = PreferredVersions
	}
	for _, want := range clientPreferred {
		if slices.Contains(serverSupported, want) {
			return want
		}
	}
	return ""
}
