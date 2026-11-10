package modern

import (
	"bufio"
	"bytes"
	"encoding/json/v2"
	"io"
	"slices"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// Event is one server-sent event from a request-scoped SSE response.
type Event struct {
	ID   string
	Name string
	Data []byte
}

func decodeResponse(contentType string, r io.Reader) ([]byte, []Event, error) {
	if strings.Contains(contentType, "text/event-stream") {
		var events []Event
		err := readSSE(r, func(ev Event) error {
			events = append(events, ev)
			return nil
		})
		if err != nil {
			return nil, events, errclass.Protocol{Err: err}
		}
		body := finalRPCBody(events)
		if len(body) == 0 {
			return nil, events, errclass.Protocol{Err: errEmptySSE}
		}
		return body, events, nil
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, wrapRoundTrip(err)
	}
	return body, nil, nil
}

func finalRPCBody(events []Event) []byte {
	var last []byte
	for _, ev := range events {
		if len(ev.Data) == 0 {
			continue
		}
		var probe struct {
			ID      any    `json:"id"`
			Result  any    `json:"result"`
			Error   any    `json:"error"`
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(ev.Data, &probe); err != nil || probe.JSONRPC == "" {
			continue
		}
		if probe.ID != nil || probe.Result != nil || probe.Error != nil {
			last = ev.Data
		}
	}
	if last != nil {
		return last
	}
	for _, ev := range slices.Backward(events) {
		if len(ev.Data) > 0 {
			return ev.Data
		}
	}
	return nil
}

var errEmptySSE = io.ErrUnexpectedEOF

func readSSE(r io.Reader, emit func(Event) error) error {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	var ev Event
	var data [][]byte
	flush := func() error {
		if ev.Name == "" && len(data) == 0 {
			return nil
		}
		ev.Data = bytes.Join(data, []byte{'\n'})
		err := emit(ev)
		ev = Event{ID: ev.ID}
		data = nil
		return err
	}
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "id":
			ev.ID = value
		case "event":
			ev.Name = value
		case "data":
			data = append(data, []byte(value))
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return flush()
}
