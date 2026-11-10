package diagnostic

// AttachChildStderr stores bounded child-process stderr as diagnostic data.
// It never classifies the record as a protocol failure.
func AttachChildStderr(rec Record, stderr string) Record {
	rec.ChildStderr = stderr
	if rec.Code == "" {
		rec.Code = CodeChildStderr
	}
	if rec.Severity == "" {
		rec.Severity = SeverityInfo
	}
	return rec
}
