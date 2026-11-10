package replay

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/phillarmonic/mcpdev/internal/clock"
	"github.com/phillarmonic/mcpdev/internal/errclass"
	"github.com/phillarmonic/mcpdev/internal/port"
)

// Stream is a replayed NDJSON pair: Writer is client-to-server, Reader is server-to-client.
type Stream struct {
	Reader io.ReadCloser
	Writer io.WriteCloser
	Events []Delivery
}

// OpenStream plays wire steps against JSON-RPC lines. It does not import the MCP SDK.
func OpenStream(ctx context.Context, doc Document, clk port.Clock) (*Stream, error) {
	if err := validate(doc); err != nil {
		return nil, err
	}
	if clk == nil {
		clk = clock.Wall{}
	}
	cr, cw := io.Pipe()
	sr, sw := io.Pipe()
	out := &Stream{Reader: sr, Writer: cw}
	go runStream(ctx, doc, clk, cr, sw, out)
	return out, nil
}

func runStream(ctx context.Context, doc Document, clk port.Clock, client io.ReadCloser, server io.WriteCloser, out *Stream) {
	defer func() {
		_ = client.Close()
		_ = server.Close()
	}()
	var mu sync.Mutex
	next := 0
	sc := bufio.NewScanner(client)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var env struct {
			Params any    `json:"params"`
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(line, &env); err != nil {
			return
		}
		if strings.HasPrefix(env.Method, "notifications/") {
			if next >= len(doc.Steps) || doc.Steps[next].Expect == nil || doc.Steps[next].Expect.Method != env.Method {
				continue
			}
		}
		if next >= len(doc.Steps) {
			return
		}
		st := doc.Steps[next]
		next++
		if err := matchExpect(next, st.Expect, env.Method, env.Params); err != nil {
			return
		}
		for _, d := range st.Deliver {
			if err := waitDelay(ctx, clk, d.DelayMS); err != nil {
				return
			}
			switch d.Kind {
			case KindSSE, KindNotification:
				mu.Lock()
				out.Events = append(out.Events, d)
				mu.Unlock()
				if d.Kind == KindNotification || (d.Kind == KindSSE && len(d.Payload) > 0) {
					frame := d.Payload
					if d.Kind == KindNotification && len(frame) == 0 {
						frame, _ = json.Marshal(map[string]any{
							"jsonrpc": "2.0",
							"method":  d.Event,
						})
					}
					if len(frame) > 0 {
						_, _ = server.Write(append(append([]byte{}, frame...), '\n'))
					}
				}
			case KindMalformed:
				text := d.Text
				if text == "" {
					text = "{not json"
				}
				_, _ = server.Write([]byte(text + "\n"))
				mu.Lock()
				out.Events = append(out.Events, d)
				mu.Unlock()
			case KindProcessExit:
				mu.Lock()
				out.Events = append(out.Events, d)
				mu.Unlock()
				return
			case KindError:
				body, _ := json.Marshal(map[string]any{
					"jsonrpc": "2.0",
					"id":      env.ID,
					"error": map[string]any{
						"code":    d.Code,
						"message": d.Message,
					},
				})
				_, _ = server.Write(append(body, '\n'))
			case KindResult:
				var result any
				if len(d.Payload) > 0 {
					_ = json.Unmarshal(d.Payload, &result)
				}
				body, _ := json.Marshal(map[string]any{
					"jsonrpc": "2.0",
					"id":      env.ID,
					"result":  result,
				})
				_, _ = server.Write(append(body, '\n'))
			}
		}
	}
	_ = sc.Err()
}

// PlayDeliveries runs unsolicited prefix steps (expect == nil) for fixture tests.
func PlayDeliveries(doc Document) ([]Delivery, error) {
	if err := validate(doc); err != nil {
		return nil, err
	}
	var out []Delivery
	for i, st := range doc.Steps {
		if st.Expect != nil {
			continue
		}
		for _, d := range st.Deliver {
			switch d.Kind {
			case KindSSE, KindMalformed, KindProcessExit, KindNotification:
				out = append(out, d)
			default:
				return out, errclass.Usage{Msg: fmt.Sprintf("replay: step %d unsolicited deliver must be sse, malformed, notification, or process_exit", i+1)}
			}
		}
	}
	return out, nil
}
