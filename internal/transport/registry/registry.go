package registry

import (
	"slices"
	"sync"

	"github.com/phillarmonic/mcpdev/internal/errclass"
)

// Registry maps connector names to factories. It does not retain live Handles.
type Registry struct {
	desc      map[string]Descriptor
	factories map[string]Factory
	mu        sync.RWMutex
}

// New is an empty registry. Use Builtins for stdio and Streamable HTTP.
func New() *Registry {
	return &Registry{
		desc:      make(map[string]Descriptor),
		factories: make(map[string]Factory),
	}
}

// Builtins returns a registry with stdio, http, and explicit HTTP era names.
func Builtins() *Registry {
	r := New()
	r.mustRegister(stdioDescriptor(), openStdio)
	r.mustRegister(httpDescriptor(), openHTTP)
	r.mustRegister(http2025Descriptor(), openHTTP)
	r.mustRegister(http2026Descriptor(), openHTTP)
	return r
}

func (r *Registry) mustRegister(d Descriptor, f Factory) {
	if err := r.Register(d, f); err != nil {
		panic(err)
	}
}

// Register adds a connector. Duplicate names and incomplete semantics fail.
func (r *Registry) Register(d Descriptor, f Factory) error {
	if f == nil {
		return errclass.Usage{Msg: "connector: factory is required"}
	}
	if err := d.validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.desc[d.Name]; ok {
		return errclass.Usage{Msg: "connector: duplicate name " + d.Name}
	}
	r.desc[d.Name] = d
	r.factories[d.Name] = f
	return nil
}

// Lookup returns a registered descriptor.
func (r *Registry) Lookup(name string) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.desc[name]
	return d, ok
}

// Names returns registered names in sorted order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.desc))
	for n := range r.desc {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

// DefaultEnums are the names that CLI help may list: stdio and http.
func DefaultEnums(r *Registry) []string {
	var out []string
	for _, n := range []string{NameStdio, NameHTTP} {
		if _, ok := r.Lookup(n); ok {
			out = append(out, n)
		}
	}
	return out
}
