package diff

import (
	"sort"
	"strings"
)

// StrSlice records a diff for an ordered []string field. Both sides are
// pointer-typed. When the lengths match, per-index entries are emitted only
// for the differing positions; when they don't match, a single whole-slice
// diff is emitted.
func (b *Builder) StrSlice(path string, exp, act *[]string) {
	if exp == nil && act == nil {
		return
	}
	if exp == nil || act == nil {
		b.Add(path, formatStrSlice(exp), formatStrSlice(act))
		return
	}
	if len(*exp) != len(*act) {
		b.Add(path, formatStrSlice(exp), formatStrSlice(act))
		return
	}
	for i := range *exp {
		if (*exp)[i] != (*act)[i] {
			b.Sub(path).Index(i).Str("", new((*exp)[i]), new((*act)[i]))
		}
	}
}

// StrSliceUnordered records a single diff entry if the two slices are not
// set-equal (ignoring order and duplicates).
func (b *Builder) StrSliceUnordered(path string, exp, act *[]string) {
	if exp == nil && act == nil {
		return
	}
	if exp == nil || act == nil {
		b.Add(path, formatStrSlice(exp), formatStrSlice(act))
		return
	}
	if equalStrSet(*exp, *act) {
		return
	}
	b.Add(path, formatStrSliceSorted(*exp), formatStrSliceSorted(*act))
}

func formatStrSlice(p *[]string) string {
	if p == nil {
		return NilSentinel
	}
	return "[" + strings.Join(*p, ",") + "]"
}

func formatStrSliceSorted(s []string) string {
	cp := make([]string, len(s))
	copy(cp, s)
	sort.Strings(cp)
	return "[" + strings.Join(cp, ",") + "]"
}

func equalStrSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, v := range a {
		m[v]++
	}
	for _, v := range b {
		m[v]--
		if m[v] < 0 {
			return false
		}
	}
	return true
}
