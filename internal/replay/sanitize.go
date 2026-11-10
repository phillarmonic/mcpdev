package replay

import (
	jsonv2 "encoding/json/v2"

	"github.com/phillarmonic/mcpdev/internal/redact"
)

// Sanitize redacts credentials in every params, metadata, and payload object.
func Sanitize(doc Document, extra []string) Document {
	o := redact.Options{ExtraKeys: extra}
	out := doc
	out.Steps = make([]Step, len(doc.Steps))
	for i, st := range doc.Steps {
		ns := st
		if st.Expect != nil {
			exp := *st.Expect
			exp.Params = redactRaw(exp.Params, o)
			exp.Meta = redactRaw(exp.Meta, o)
			ns.Expect = &exp
		}
		ns.Deliver = make([]Delivery, len(st.Deliver))
		for j, d := range st.Deliver {
			d.Payload = redactRaw(d.Payload, o)
			d.Text = redact.Text(d.Text, o)
			d.Message = redact.Text(d.Message, o)
			ns.Deliver[j] = d
		}
		out.Steps[i] = ns
	}
	return out
}

func redactRaw(raw []byte, o redact.Options) []byte {
	if len(raw) == 0 {
		return raw
	}
	var v any
	if err := jsonv2.Unmarshal(raw, &v); err != nil {
		return []byte(redact.Text(string(raw), o))
	}
	b, err := jsonv2.Marshal(redact.Any(v, o), jsonv2.Deterministic(true))
	if err != nil {
		return raw
	}
	return b
}
