package trace

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"strings"
)

type jsonEvent struct {
	Headers      map[string]string `json:"headers,omitempty"`
	Kind         string            `json:"kind"`
	Direction    string            `json:"direction,omitempty"`
	Transport    string            `json:"transport,omitempty"`
	Method       string            `json:"method,omitempty"`
	RequestID    string            `json:"request_id,omitempty"`
	Note         string            `json:"note,omitempty"`
	Payload      string            `json:"payload,omitempty"`
	Seq          int               `json:"seq"`
	HTTPStatus   int               `json:"http_status,omitempty"`
	Attempt      int               `json:"attempt,omitempty"`
	PayloadBytes int               `json:"payload_bytes,omitempty"`
	ElapsedMS    int64             `json:"elapsed_ms"`
	Truncated    bool              `json:"truncated,omitempty"`
}

func writeOne(w io.Writer, ev Event, format Format) error {
	if format == FormatJSON {
		b, err := json.Marshal(toJSON(ev), json.Deterministic(true))
		if err != nil {
			return err
		}
		_, err = w.Write(append(b, '\n'))
		return err
	}
	arrow := "-"
	switch ev.Direction {
	case DirClient:
		arrow = "→"
	case DirServer:
		arrow = "←"
	case DirInternal:
		arrow = "·"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#%03d +%dms %s %s", ev.Seq, ev.Elapsed.Milliseconds(), arrow, ev.Kind)
	if ev.Transport != "" {
		fmt.Fprintf(&b, " %s", ev.Transport)
	}
	if ev.HTTPStatus != 0 {
		fmt.Fprintf(&b, " %d", ev.HTTPStatus)
	}
	if ev.Method != "" {
		fmt.Fprintf(&b, " %s", ev.Method)
	}
	if ev.RequestID != "" {
		fmt.Fprintf(&b, " id=%s", ev.RequestID)
	}
	if ev.Attempt != 0 {
		fmt.Fprintf(&b, " attempt=%d", ev.Attempt)
	}
	if ev.Note != "" {
		fmt.Fprintf(&b, " %s", ev.Note)
	}
	if ev.Truncated {
		b.WriteString(" truncated")
	}
	b.WriteByte('\n')
	if ev.Payload != "" {
		b.WriteString("  ")
		b.WriteString(ev.Payload)
		b.WriteByte('\n')
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func toJSON(ev Event) jsonEvent {
	return jsonEvent{
		Seq:          ev.Seq,
		ElapsedMS:    ev.Elapsed.Milliseconds(),
		Kind:         string(ev.Kind),
		Direction:    string(ev.Direction),
		Transport:    ev.Transport,
		Method:       ev.Method,
		RequestID:    ev.RequestID,
		HTTPStatus:   ev.HTTPStatus,
		Attempt:      ev.Attempt,
		Payload:      ev.Payload,
		PayloadBytes: ev.PayloadBytes,
		Truncated:    ev.Truncated,
		Headers:      ev.Headers,
		Note:         ev.Note,
	}
}
