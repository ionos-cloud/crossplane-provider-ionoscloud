package diff

import (
	"strconv"
	"strings"
)

// NilSentinel is the rendered placeholder for a nil pointer when the
// counterpart side is non-nil.
const NilSentinel = "<nil>"

// entrySeparator joins individual diff entries in [Builder.String].
const entrySeparator = "; "

// entry is one recorded diff line.
type entry struct {
	path     string
	expected string
	actual   string
}

// Builder accumulates pre-formatted diff entries. The zero value is unusable;
// construct with [New]. Sub-builders created via [Builder.Sub] or
// [Builder.Index] share the same underlying entry slice as the root, so
// recording through a child affects the root's result.
type Builder struct {
	prefix  string
	entries *[]entry
}

// New returns a fresh top-level Builder.
func New() *Builder {
	e := make([]entry, 0)
	return &Builder{entries: &e}
}

// Sub returns a child Builder whose recorded paths are prefixed with name
// joined by a dot. Use it to scope a nested struct:
//
//	mw := d.Sub("maintenanceWindow")
//	mw.Str("time", &exp.Time, act.Time)
func (b *Builder) Sub(name string) *Builder {
	return &Builder{prefix: joinPath(b.prefix, name), entries: b.entries}
}

// Index returns a child Builder whose prefix has "[i]" appended. Use it
// inside slice iteration:
//
//	for i := range nics {
//	    n := d.Sub("nics").Index(i)
//	    n.Str("name", &nics[i].Name, observed[i].Name)
//	}
func (b *Builder) Index(i int) *Builder {
	return &Builder{prefix: b.prefix + "[" + strconv.Itoa(i) + "]", entries: b.entries}
}

// Add records a raw entry. Helper functions use it after formatting their
// own values. The path is joined with the Builder's current prefix.
func (b *Builder) Add(path, expected, actual string) {
	*b.entries = append(*b.entries, entry{
		path:     joinPath(b.prefix, path),
		expected: expected,
		actual:   actual,
	})
}

// Empty reports whether no diff entries have been recorded.
func (b *Builder) Empty() bool { return len(*b.entries) == 0 }

// String renders the recorded entries as "path exp=V act=V" joined by "; ".
// Returns "" if no entries have been recorded.
func (b *Builder) String() string {
	if len(*b.entries) == 0 {
		return ""
	}
	parts := make([]string, 0, len(*b.entries))
	for _, e := range *b.entries {
		parts = append(parts, e.path+" exp="+e.expected+" act="+e.actual)
	}
	return strings.Join(parts, entrySeparator)
}

// Result returns (upToDate, diff). upToDate is true exactly when no entries
// have been recorded. Use as the canonical return for IsUpToDate functions.
func (b *Builder) Result() (bool, string) { return b.Empty(), b.String() }

func joinPath(prefix, name string) string {
	switch {
	case prefix == "":
		return name
	case name == "":
		return prefix
	case strings.HasPrefix(name, "["):
		return prefix + name
	default:
		return prefix + "." + name
	}
}
