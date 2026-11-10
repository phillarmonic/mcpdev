package testkit

import (
	"fmt"
	"slices"
)

// Names returns catalog keys in stable order.
func Names() []string {
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Named returns a catalog fixture. Unknown names are an error, not a silent default.
func Named(name string) (Descriptor, error) {
	d, ok := catalog[name]
	if !ok {
		return Descriptor{}, fmt.Errorf("testkit: unknown fixture %q", name)
	}
	d.Name = name
	return d, nil
}

var catalog = map[string]Descriptor{
	"legacy-stdio": {
		Era:              Era20251125,
		Transport:        TransportStdio,
		Capabilities:     fullCaps(true),
		RequireHandshake: true,
		Instructions:     "mcpdev fixture laboratory",
	},
	"legacy-http": {
		Era:              Era20251125,
		Transport:        TransportHTTP,
		Capabilities:     fullCaps(true),
		RequireHandshake: true,
		Session:          SessionPolicy{MintID: true, ID: "fixture-session"},
	},
	"legacy-paginate": {
		Era:              Era20251125,
		Transport:        TransportStdio,
		Capabilities:     fullCaps(false),
		RequireHandshake: true,
		PageSize:         2,
		ExtraTools:       4,
	},
	"legacy-notify": {
		Era:               Era20251125,
		Transport:         TransportStdio,
		Capabilities:      fullCaps(true),
		RequireHandshake:  true,
		NotifyListChanged: true,
	},
	"modern-stdio": {
		Era:          Era20260728,
		Transport:    TransportStdio,
		Capabilities: fullCaps(true),
	},
	"modern-http": {
		Era:          Era20260728,
		Transport:    TransportHTTP,
		Capabilities: fullCaps(true),
	},
	"modern-sse": {
		Era:          Era20260728,
		Transport:    TransportHTTP,
		Capabilities: fullCaps(false),
		SSEOnCall:    true,
	},
	"modern-auth": {
		Era:          Era20260728,
		Transport:    TransportHTTP,
		Capabilities: fullCaps(false),
		AuthRequired: true,
		AuthToken:    "fixture-token",
	},
	"modern-need-cap": {
		Era:          Era20260728,
		Transport:    TransportStdio,
		Capabilities: fullCaps(false),
		Malformed:    MalformedMissingClientCap,
	},
	"modern-version-retry": {
		Era:          Era20260728,
		Transport:    TransportStdio,
		Capabilities: fullCaps(true),
		Malformed:    MalformedModernUnsupported,
	},
	"delay-stdio": {
		Era:              Era20251125,
		Transport:        TransportStdio,
		Capabilities:     fullCaps(false),
		RequireHandshake: true,
		Delay:            timeDelay,
	},
	"malformed-stdio": {
		Era:       Era20251125,
		Transport: TransportStdio,
		Malformed: MalformedJSON,
	},
	"malformed-http": {
		Era:       Era20260728,
		Transport: TransportHTTP,
		Malformed: MalformedEmptyHTTPBody,
	},
	"unrecognized-http": {
		Era:       Era20251125,
		Transport: TransportHTTP,
		Malformed: MalformedUnrecognizedError,
	},
	"legacy-unsupported": {
		Era:              Era20251125,
		Transport:        TransportStdio,
		RequireHandshake: true,
		Malformed:        MalformedUnsupportedVersion,
	},
	"legacy-bad-caps": {
		Era:              Era20251125,
		Transport:        TransportStdio,
		RequireHandshake: true,
		Malformed:        MalformedCapabilityObject,
	},
	"legacy-exit": {
		Era:              Era20251125,
		Transport:        TransportStdio,
		RequireHandshake: true,
		Malformed:        MalformedExitOnFirstRPC,
	},
}

func fullCaps(listChanged bool) Capabilities {
	return Capabilities{
		Tools:             true,
		Prompts:           true,
		Resources:         true,
		ListChanged:       listChanged,
		ResourceSubscribe: listChanged,
	}
}
