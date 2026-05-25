package diff

import "testing"

func TestNewEmptyResult(t *testing.T) {
	d := New()
	if !d.Empty() {
		t.Fatalf("new builder should be empty")
	}
	if d.String() != "" {
		t.Fatalf("empty builder String should be empty, got %q", d.String())
	}
	ok, s := d.Result()
	if !ok || s != "" {
		t.Fatalf("empty Result should be (true, \"\"), got (%v, %q)", ok, s)
	}
}

func TestAddAndJoin(t *testing.T) {
	d := New()
	d.Add("a", "1", "2")
	d.Add("b", "x", "y")
	want := "a exp=1 act=2; b exp=x act=y"
	if got := d.String(); got != want {
		t.Fatalf("String mismatch:\n want: %s\n  got: %s", want, got)
	}
	if d.Empty() {
		t.Fatal("builder should not be empty after Add")
	}
	ok, _ := d.Result()
	if ok {
		t.Fatal("Result.upToDate should be false when entries exist")
	}
}

func TestSubPrefixing(t *testing.T) {
	d := New()
	d.Sub("outer").Add("inner", "1", "2")
	if got := d.String(); got != "outer.inner exp=1 act=2" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestSubNested(t *testing.T) {
	d := New()
	d.Sub("a").Sub("b").Sub("c").Add("d", "x", "y")
	if got := d.String(); got != "a.b.c.d exp=x act=y" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestIndex(t *testing.T) {
	d := New()
	d.Sub("nics").Index(0).Add("ip", "1.1.1.1", "2.2.2.2")
	d.Sub("nics").Index(12).Sub("inner").Add("name", "a", "b")
	want := "nics[0].ip exp=1.1.1.1 act=2.2.2.2; nics[12].inner.name exp=a act=b"
	if got := d.String(); got != want {
		t.Fatalf("Index mismatch:\n want: %s\n  got: %s", want, got)
	}
}

func TestAddEmptyPath(t *testing.T) {
	d := New()
	d.Sub("root").Add("", "1", "2")
	if got := d.String(); got != "root exp=1 act=2" {
		t.Fatalf("empty path under sub: got %q", got)
	}
}

func TestSharedEntriesAcrossSubs(t *testing.T) {
	root := New()
	a := root.Sub("a")
	b := root.Sub("b")
	a.Add("x", "1", "2")
	b.Add("y", "3", "4")
	if got := root.String(); got != "a.x exp=1 act=2; b.y exp=3 act=4" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestJoinPathBracket(t *testing.T) {
	if got := joinPath("a", "[0]"); got != "a[0]" {
		t.Fatalf("got %q", got)
	}
}
