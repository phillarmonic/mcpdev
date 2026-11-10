package redact

import "net/url"

// URLString redacts userinfo and sensitive query names. Non-URLs are returned unchanged.
func URLString(raw string, o Options) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	if u.User != nil {
		u.User = url.User(o.replacement())
	}
	q := u.Query()
	changed := false
	for k, vs := range q {
		if !Sensitive(k, o.ExtraKeys) {
			continue
		}
		for i := range vs {
			vs[i] = o.replacement()
		}
		q[k] = vs
		changed = true
	}
	if changed {
		u.RawQuery = q.Encode()
	}
	return u.String()
}
