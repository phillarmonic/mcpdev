package legacy

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// Event is one server-sent event from POST or GET SSE.
type Event struct {
	ID   string
	Name string
	Data []byte
}

func decodePOSTBody(contentType string, body []byte) ([]byte, []Event, error) {
	if strings.Contains(contentType, "text/event-stream") {
		var events []Event
		err := readSSE(bytes.NewReader(body), func(ev Event) error {
			events = append(events, ev)
			return nil
		})
		if err != nil {
			return nil, events, errclass.Protocol{Err: err}
		}
		if len(events) == 0 || len(events[0].Data) == 0 {
			return nil, events, errclass.Protocol{Err: errEmptySSE}
		}
		return events[0].Data, events, nil
	}
	return body, nil, nil
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
