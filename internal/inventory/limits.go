package inventory

// Limits bound cursor walking. Zero values become the defaults so collection cannot run unbounded.
type Limits struct {
	MaxPages int
	MaxItems int
}

const (
	defaultMaxPages = 256
	defaultMaxItems = 10_000
)

func (l Limits) resolved() Limits {
	if l.MaxPages <= 0 {
		l.MaxPages = defaultMaxPages
	}
	if l.MaxItems <= 0 {
		l.MaxItems = defaultMaxItems
	}
	return l
}
