package utils

import "testing"

func Test_PointerString(t *testing.T) {
	p := PointerString("bla")
	if p == nil {
		t.Fail()
		return
	}
	if *p != "bla" {
		t.Fail()
	}
}
