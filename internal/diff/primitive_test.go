package diff

import "testing"

func TestStr(t *testing.T) {
	cases := []struct {
		name     string
		exp, act *string
		want     string
	}{
		{"both nil", nil, nil, ""},
		{"equal", new("a"), new("a"), ""},
		{"different", new("a"), new("b"), "f exp=a act=b"},
		{"exp nil", nil, new("b"), "f exp=<nil> act=b"},
		{"act nil", new("a"), nil, "f exp=a act=<nil>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := New()
			d.Str("f", c.exp, c.act)
			if got := d.String(); got != c.want {
				t.Errorf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestInt(t *testing.T) {
	cases := []struct {
		name     string
		exp, act *int
		want     string
	}{
		{"both nil", nil, nil, ""},
		{"equal", new(1), new(1), ""},
		{"different", new(1), new(2), "f exp=1 act=2"},
		{"exp nil", nil, new(3), "f exp=<nil> act=3"},
		{"act nil", new(3), nil, "f exp=3 act=<nil>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := New()
			d.Int("f", c.exp, c.act)
			if got := d.String(); got != c.want {
				t.Errorf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestInt32(t *testing.T) {
	d := New()
	d.Int32("f", new(int32(4)), new(int32(5)))
	if got := d.String(); got != "f exp=4 act=5" {
		t.Errorf("got %q", got)
	}
	d = New()
	d.Int32("f", nil, nil)
	if !d.Empty() {
		t.Error("nil/nil should be empty")
	}
	d = New()
	d.Int32("f", new(int32(1)), new(int32(1)))
	if !d.Empty() {
		t.Error("equal should be empty")
	}
	d = New()
	d.Int32("f", nil, new(int32(2)))
	if got := d.String(); got != "f exp=<nil> act=2" {
		t.Errorf("got %q", got)
	}
	d = New()
	d.Int32("f", new(int32(2)), nil)
	if got := d.String(); got != "f exp=2 act=<nil>" {
		t.Errorf("got %q", got)
	}
}

func TestInt64(t *testing.T) {
	d := New()
	d.Int64("f", new(int64(10)), new(int64(20)))
	if got := d.String(); got != "f exp=10 act=20" {
		t.Errorf("got %q", got)
	}
	d = New()
	d.Int64("f", nil, nil)
	if !d.Empty() {
		t.Error("nil/nil empty")
	}
	d = New()
	d.Int64("f", new(int64(7)), new(int64(7)))
	if !d.Empty() {
		t.Error("equal empty")
	}
	d = New()
	d.Int64("f", nil, new(int64(1)))
	if d.Empty() {
		t.Error("one nil should diff")
	}
	d = New()
	d.Int64("f", new(int64(1)), nil)
	if d.Empty() {
		t.Error("one nil should diff")
	}
}

func TestFloat32(t *testing.T) {
	d := New()
	d.Float32("f", new(float32(1.5)), new(float32(2.5)))
	if got := d.String(); got != "f exp=1.5 act=2.5" {
		t.Errorf("got %q", got)
	}
	d = New()
	d.Float32("f", nil, nil)
	if !d.Empty() {
		t.Error("nil/nil empty")
	}
	d = New()
	d.Float32("f", new(float32(1.5)), new(float32(1.5)))
	if !d.Empty() {
		t.Error("equal empty")
	}
	d = New()
	d.Float32("f", nil, new(float32(1)))
	if d.Empty() {
		t.Error("one nil diff")
	}
	d = New()
	d.Float32("f", new(float32(1)), nil)
	if d.Empty() {
		t.Error("one nil diff")
	}
}

func TestFloat64(t *testing.T) {
	d := New()
	d.Float64("f", new(3.14), new(2.71))
	if got := d.String(); got != "f exp=3.14 act=2.71" {
		t.Errorf("got %q", got)
	}
	d = New()
	d.Float64("f", nil, nil)
	if !d.Empty() {
		t.Error("nil/nil empty")
	}
	d = New()
	d.Float64("f", new(1.0), new(1.0))
	if !d.Empty() {
		t.Error("equal empty")
	}
	d = New()
	d.Float64("f", nil, new(1.0))
	if d.Empty() {
		t.Error("one nil diff")
	}
	d = New()
	d.Float64("f", new(1.0), nil)
	if d.Empty() {
		t.Error("one nil diff")
	}
}

func TestBool(t *testing.T) {
	cases := []struct {
		name     string
		exp, act *bool
		want     string
	}{
		{"both nil", nil, nil, ""},
		{"equal true", new(true), new(true), ""},
		{"equal false", new(false), new(false), ""},
		{"different", new(true), new(false), "f exp=true act=false"},
		{"exp nil", nil, new(false), "f exp=<nil> act=false"},
		{"act nil", new(true), nil, "f exp=true act=<nil>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := New()
			d.Bool("f", c.exp, c.act)
			if got := d.String(); got != c.want {
				t.Errorf("got %q want %q", got, c.want)
			}
		})
	}
}
