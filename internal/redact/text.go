package redact

import "strings"

// Text redacts header-shaped lines and URL credentials in free-form diagnostics.
func Text(s string, o Options) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = textLine(line, o)
	}
	return strings.Join(lines, "\n")
}

func textLine(line string, o Options) string {
	if name, _, ok := strings.Cut(line, ":"); ok && Sensitive(strings.TrimSpace(name), o.ExtraKeys) {
		return redactHeaderLine(line, o)
	}
	fields := strings.Fields(line)
	changed := false
	for i, f := range fields {
		n := URLString(strings.Trim(f, "<>\"'"), o)
		if n != strings.Trim(f, "<>\"'") {
			fields[i] = n
			changed = true
		}
	}
	if changed {
		return strings.Join(fields, " ")
	}
	return URLString(line, o)
}
