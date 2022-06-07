package crds

import (
	"testing"
)

func TestMustGetCRDs(t *testing.T) {
	MustGetCRDs() // doesn't panic
}

func TestGetCRDs(t *testing.T) {
	_, err := GetCRDs()
	if err != nil {
		t.Error(err)
	}
}
