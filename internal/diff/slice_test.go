package diff

import "testing"

func TestStrSliceBothNil(t *testing.T) {
	d := New()
	d.StrSlice("ips", nil, nil)
	if !d.Empty() {
		t.Fatalf("got %q", d.String())
	}
}

func TestStrSliceOneNil(t *testing.T) {
	d := New()
	v := []string{"a", "b"}
	d.StrSlice("ips", nil, &v)
	if got := d.String(); got != "ips exp=<nil> act=[a,b]" {
		t.Fatalf("got %q", got)
	}
	d = New()
	d.StrSlice("ips", &v, nil)
	if got := d.String(); got != "ips exp=[a,b] act=<nil>" {
		t.Fatalf("got %q", got)
	}
}

func TestStrSliceLengthMismatch(t *testing.T) {
	d := New()
	a := []string{"a", "b"}
	b := []string{"a"}
	d.StrSlice("ips", &a, &b)
	if got := d.String(); got != "ips exp=[a,b] act=[a]" {
		t.Fatalf("got %q", got)
	}
}

func TestStrSlicePerIndex(t *testing.T) {
	d := New()
	a := []string{"a", "b", "c"}
	b := []string{"a", "B", "C"}
	d.StrSlice("ips", &a, &b)
	want := "ips[1] exp=b act=B; ips[2] exp=c act=C"
	if got := d.String(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStrSliceEqual(t *testing.T) {
	d := New()
	a := []string{"a", "b"}
	b := []string{"a", "b"}
	d.StrSlice("ips", &a, &b)
	if !d.Empty() {
		t.Fatalf("equal slices: got %q", d.String())
	}
}

func TestStrSliceUnordered(t *testing.T) {
	d := New()
	a := []string{"b", "a"}
	b := []string{"a", "b"}
	d.StrSliceUnordered("ips", &a, &b)
	if !d.Empty() {
		t.Fatalf("set-equal: got %q", d.String())
	}
}

func TestStrSliceUnorderedDiffers(t *testing.T) {
	d := New()
	a := []string{"a", "b"}
	b := []string{"a", "c"}
	d.StrSliceUnordered("ips", &a, &b)
	if got := d.String(); got != "ips exp=[a,b] act=[a,c]" {
		t.Fatalf("got %q", got)
	}
}

func TestStrSliceUnorderedLengths(t *testing.T) {
	d := New()
	a := []string{"a"}
	b := []string{"a", "b"}
	d.StrSliceUnordered("ips", &a, &b)
	if got := d.String(); got != "ips exp=[a] act=[a,b]" {
		t.Fatalf("got %q", got)
	}
}

func TestStrSliceUnorderedDuplicates(t *testing.T) {
	d := New()
	a := []string{"a", "a", "b"}
	b := []string{"a", "b", "b"}
	d.StrSliceUnordered("ips", &a, &b)
	if d.Empty() {
		t.Fatal("multiset semantics: a×2,b×1 != a×1,b×2 should diff")
	}
}

func TestStrSliceUnorderedBothNil(t *testing.T) {
	d := New()
	d.StrSliceUnordered("ips", nil, nil)
	if !d.Empty() {
		t.Fatalf("got %q", d.String())
	}
}

func TestStrSliceUnorderedOneNil(t *testing.T) {
	d := New()
	v := []string{"a"}
	d.StrSliceUnordered("ips", nil, &v)
	if got := d.String(); got != "ips exp=<nil> act=[a]" {
		t.Fatalf("got %q", got)
	}
	d = New()
	d.StrSliceUnordered("ips", &v, nil)
	if got := d.String(); got != "ips exp=[a] act=<nil>" {
		t.Fatalf("got %q", got)
	}
}
