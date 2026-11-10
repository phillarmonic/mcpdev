package registry

import "github.com/phillarmonic/mcpdev/internal/extension"

func stdioDescriptor() Descriptor {
	return Descriptor{
		Name:     NameStdio,
		Kind:     KindBuiltin,
		Standard: true,
		Semantics: extension.Semantics{
			Cancellation: extension.CancelSharedNotification,
			Streaming:    extension.StreamNewlineJSON,
			Metadata:     extension.MetaPerRequest,
			Credentials:  extension.CredEnv,
		},
	}
}

func httpDescriptor() Descriptor {
	return Descriptor{
		Name:     NameHTTP,
		Kind:     KindBuiltin,
		Standard: true,
		Semantics: extension.Semantics{
			Cancellation: extension.CancelRequestContext,
			Streaming:    extension.StreamRequestSSE,
			Metadata:     extension.MetaPerRequest,
			Credentials:  extension.CredHeaders,
		},
	}
}

func http2025Descriptor() Descriptor {
	d := httpDescriptor()
	d.Name = NameHTTP2025
	d.Semantics.Metadata = extension.MetaInitialize
	return d
}

func http2026Descriptor() Descriptor {
	d := httpDescriptor()
	d.Name = NameHTTP2026
	return d
}
