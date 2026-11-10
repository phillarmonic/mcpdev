package trace

import (
	"bytes"
	"io"
)

// WrapReader emits one server frame per NDJSON line. Disabled recorders return r unchanged.
func WrapReader(r io.ReadCloser, rec *Recorder, transport string) io.ReadCloser {
	if r == nil || !rec.Enabled() {
		return r
	}
	return &lineReader{inner: r, rec: rec, transport: transport}
}

// WrapWriter emits one client frame per NDJSON line. Disabled recorders return w unchanged.
func WrapWriter(w io.WriteCloser, rec *Recorder, transport string) io.WriteCloser {
	if w == nil || !rec.Enabled() {
		return w
	}
	return &lineWriter{inner: w, rec: rec, transport: transport}
}

type lineReader struct {
	inner     io.ReadCloser
	rec       *Recorder
	transport string
	buf       bytes.Buffer
}

func (r *lineReader) Read(p []byte) (int, error) {
	n, err := r.inner.Read(p)
	if n > 0 {
		r.consume(p[:n])
	}
	if err != nil && r.buf.Len() > 0 {
		r.flushRest()
	}
	return n, err
}

func (r *lineReader) consume(p []byte) {
	for _, b := range p {
		if b == '\n' {
			r.rec.frame(DirServer, r.transport, bytes.TrimSpace(r.buf.Bytes()))
			r.buf.Reset()
			continue
		}
		_ = r.buf.WriteByte(b)
	}
}

func (r *lineReader) flushRest() {
	line := bytes.TrimSpace(r.buf.Bytes())
	r.buf.Reset()
	if len(line) > 0 {
		r.rec.frame(DirServer, r.transport, line)
	}
}

func (r *lineReader) Close() error {
	if r.buf.Len() > 0 {
		r.flushRest()
	}
	return r.inner.Close()
}

type lineWriter struct {
	inner     io.WriteCloser
	rec       *Recorder
	transport string
	buf       bytes.Buffer
}

func (w *lineWriter) Write(p []byte) (int, error) {
	n, err := w.inner.Write(p)
	if n > 0 {
		w.consume(p[:n])
	}
	return n, err
}

func (w *lineWriter) consume(p []byte) {
	for _, b := range p {
		if b == '\n' {
			w.rec.frame(DirClient, w.transport, bytes.TrimSpace(w.buf.Bytes()))
			w.buf.Reset()
			continue
		}
		_ = w.buf.WriteByte(b)
	}
}

func (w *lineWriter) Close() error {
	if w.buf.Len() > 0 {
		line := bytes.TrimSpace(w.buf.Bytes())
		w.buf.Reset()
		if len(line) > 0 {
			w.rec.frame(DirClient, w.transport, line)
		}
	}
	return w.inner.Close()
}
