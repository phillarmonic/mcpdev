package replay

import "encoding/json"

const (
	// Kind is the document type recorded in fixtures.
	Kind = "mcpdev.replay"
	// SchemaVersion is the current on-disk schema.
	SchemaVersion = 1
)

// Document is a versioned replay script.
type Document struct {
	Meta          Meta        `json:"meta"`
	Session       SessionSpec `json:"session"`
	Kind          string      `json:"kind"`
	Steps         []Step      `json:"steps"`
	SchemaVersion int         `json:"schema_version"`
}

// Meta is fixture provenance. It is not a live Target.
type Meta struct {
	Transport string `json:"transport,omitzero"`
	Era       string `json:"era,omitzero"`
	Protocol  string `json:"protocol_version,omitzero"`
	Note      string `json:"note,omitzero"`
}

// SessionSpec is negotiated session metadata restored by the replay connector.
type SessionSpec struct {
	Capabilities    map[string]bool `json:"capabilities,omitzero"`
	Era             string          `json:"era,omitzero"`
	Handshake       string          `json:"handshake,omitzero"`
	DiscoverState   string          `json:"discover_state,omitzero"`
	ProtocolVersion string          `json:"protocol_version,omitzero"`
	ServerName      string          `json:"server_name,omitzero"`
	ServerTitle     string          `json:"server_title,omitzero"`
	ServerVersion   string          `json:"server_version,omitzero"`
	SessionID       string          `json:"session_id,omitzero"`
	Instructions    string          `json:"instructions,omitzero"`
	FallbackReason  string          `json:"fallback_reason,omitzero"`
	Fallback        bool            `json:"fallback,omitzero"`
}

// Step is one expected client request plus ordered server deliveries.
type Step struct {
	Expect  *Expect    `json:"expect,omitempty"`
	Deliver []Delivery `json:"deliver"`
}

// Expect matches a client JSON-RPC method, params, and optional _meta.
type Expect struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitzero"`
	Meta   json.RawMessage `json:"_meta,omitzero"`
}

// Delivery kinds.
const (
	KindResult       = "result"
	KindError        = "error"
	KindNotification = "notification"
	KindSSE          = "sse"
	KindMalformed    = "malformed"
	KindProcessExit  = "process_exit"
)

// Delivery is one server action after a matched request (or an unsolicited event).
type Delivery struct {
	Kind     string          `json:"kind"`
	Event    string          `json:"event,omitzero"`
	ID       string          `json:"id,omitzero"`
	Text     string          `json:"text,omitzero"`
	Message  string          `json:"message,omitzero"`
	Category string          `json:"category,omitzero"`
	Payload  json.RawMessage `json:"payload,omitzero"`
	DelayMS  int             `json:"delay_ms,omitzero"`
	Code     int             `json:"code,omitzero"`
}
