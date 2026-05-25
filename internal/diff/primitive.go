package diff

import "strconv"

// Str records a diff for a string field. Both arguments are pointers; nil
// renders as [NilSentinel]. No entry is recorded when both are nil or when
// both point to equal values.
func (b *Builder) Str(path string, exp, act *string) {
	if exp == nil && act == nil {
		return
	}
	if exp != nil && act != nil && *exp == *act {
		return
	}
	b.Add(path, derefStr(exp), derefStr(act))
}

// Int records a diff for an int field.
func (b *Builder) Int(path string, exp, act *int) {
	if exp == nil && act == nil {
		return
	}
	if exp != nil && act != nil && *exp == *act {
		return
	}
	b.Add(path, formatInt(exp), formatInt(act))
}

// Int32 records a diff for an int32 field.
func (b *Builder) Int32(path string, exp, act *int32) {
	if exp == nil && act == nil {
		return
	}
	if exp != nil && act != nil && *exp == *act {
		return
	}
	b.Add(path, formatInt32(exp), formatInt32(act))
}

// Int64 records a diff for an int64 field.
func (b *Builder) Int64(path string, exp, act *int64) {
	if exp == nil && act == nil {
		return
	}
	if exp != nil && act != nil && *exp == *act {
		return
	}
	b.Add(path, formatInt64(exp), formatInt64(act))
}

// Float32 records a diff for a float32 field.
func (b *Builder) Float32(path string, exp, act *float32) {
	if exp == nil && act == nil {
		return
	}
	if exp != nil && act != nil && *exp == *act {
		return
	}
	b.Add(path, formatFloat32(exp), formatFloat32(act))
}

// Float64 records a diff for a float64 field.
func (b *Builder) Float64(path string, exp, act *float64) {
	if exp == nil && act == nil {
		return
	}
	if exp != nil && act != nil && *exp == *act {
		return
	}
	b.Add(path, formatFloat64(exp), formatFloat64(act))
}

// Bool records a diff for a bool field.
func (b *Builder) Bool(path string, exp, act *bool) {
	if exp == nil && act == nil {
		return
	}
	if exp != nil && act != nil && *exp == *act {
		return
	}
	b.Add(path, formatBool(exp), formatBool(act))
}

func derefStr(p *string) string {
	if p == nil {
		return NilSentinel
	}
	return *p
}

func formatInt(p *int) string {
	if p == nil {
		return NilSentinel
	}
	return strconv.Itoa(*p)
}

func formatInt32(p *int32) string {
	if p == nil {
		return NilSentinel
	}
	return strconv.FormatInt(int64(*p), 10)
}

func formatInt64(p *int64) string {
	if p == nil {
		return NilSentinel
	}
	return strconv.FormatInt(*p, 10)
}

func formatFloat32(p *float32) string {
	if p == nil {
		return NilSentinel
	}
	return strconv.FormatFloat(float64(*p), 'g', -1, 32)
}

func formatFloat64(p *float64) string {
	if p == nil {
		return NilSentinel
	}
	return strconv.FormatFloat(*p, 'g', -1, 64)
}

func formatBool(p *bool) string {
	if p == nil {
		return NilSentinel
	}
	return strconv.FormatBool(*p)
}
