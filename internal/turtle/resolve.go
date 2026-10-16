package turtle

import "strings"

// isAbsolute reports whether iri has a scheme: a non-empty prefix before the
// first ':' that holds none of '/', '?', '#'.
func isAbsolute(iri string) bool {
	colon := strings.IndexByte(iri, ':')
	return colon > 0 && !strings.ContainsAny(iri[:colon], "/?#")
}

// ref holds the five components of RFC 3986 §3, split by the regular
// expression of Appendix B. A has* flag tells an empty component from an
// absent one.
type ref struct {
	scheme, authority, path, query, fragment string
	hasAuthority, hasQuery, hasFragment      bool
}

func split(s string) ref {
	var r ref
	if i := strings.IndexByte(s, '#'); i >= 0 {
		r.fragment, r.hasFragment, s = s[i+1:], true, s[:i]
	}
	if i := strings.IndexByte(s, '?'); i >= 0 {
		r.query, r.hasQuery, s = s[i+1:], true, s[:i]
	}
	if isAbsolute(s) {
		i := strings.IndexByte(s, ':')
		r.scheme, s = s[:i], s[i+1:]
	}
	if strings.HasPrefix(s, "//") {
		s = s[2:]
		i := strings.IndexByte(s, '/')
		if i < 0 {
			i = len(s)
		}
		r.authority, r.hasAuthority, s = s[:i], true, s[i:]
	}
	r.path = s
	return r
}

func (r ref) String() string {
	var b strings.Builder
	if r.scheme != "" {
		b.WriteString(r.scheme + ":")
	}
	if r.hasAuthority {
		b.WriteString("//" + r.authority)
	}
	b.WriteString(r.path)
	if r.hasQuery {
		b.WriteString("?" + r.query)
	}
	if r.hasFragment {
		b.WriteString("#" + r.fragment)
	}
	return b.String()
}

// resolve resolves the relative reference rel against the absolute base by
// RFC 3986 §5.2.2 (strict), with §5.2.3 merge and §5.2.4 dot-segment removal.
func resolve(base, rel string) string {
	b, r := split(base), split(rel)
	var t ref
	switch {
	case r.hasAuthority:
		t = ref{authority: r.authority, hasAuthority: true, path: removeDots(r.path), query: r.query, hasQuery: r.hasQuery}
	case r.path == "":
		t = ref{authority: b.authority, hasAuthority: b.hasAuthority, path: b.path, query: b.query, hasQuery: b.hasQuery}
		if r.hasQuery {
			t.query, t.hasQuery = r.query, true
		}
	default:
		t = ref{authority: b.authority, hasAuthority: b.hasAuthority, query: r.query, hasQuery: r.hasQuery}
		if strings.HasPrefix(r.path, "/") {
			t.path = removeDots(r.path)
		} else {
			t.path = removeDots(merge(b, r.path))
		}
	}
	t.scheme = b.scheme
	t.fragment, t.hasFragment = r.fragment, r.hasFragment
	return t.String()
}

// merge is RFC 3986 §5.2.3.
func merge(b ref, path string) string {
	if b.hasAuthority && b.path == "" {
		return "/" + path
	}
	if i := strings.LastIndexByte(b.path, '/'); i >= 0 {
		return b.path[:i+1] + path
	}
	return path
}

// removeDots is RFC 3986 §5.2.4.
func removeDots(in string) string {
	var out []string // output segments, each starting with '/' unless first
	for in != "" {
		switch {
		case strings.HasPrefix(in, "../"):
			in = in[3:]
		case strings.HasPrefix(in, "./"):
			in = in[2:]
		case strings.HasPrefix(in, "/./"):
			in = in[2:]
		case in == "/.":
			in = "/"
		case strings.HasPrefix(in, "/../"):
			in = in[3:]
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		case in == "/..":
			in = "/"
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		case in == "." || in == "..":
			in = ""
		default:
			start := 0
			if in[0] == '/' {
				start = 1
			}
			i := strings.IndexByte(in[start:], '/')
			if i < 0 {
				i = len(in)
			} else {
				i += start
			}
			out = append(out, in[:i])
			in = in[i:]
		}
	}
	return strings.Join(out, "")
}
