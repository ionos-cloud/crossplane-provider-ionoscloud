package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/alecthomas/kingpin.v2"
)

func TestIsFlagSetByUser(t *testing.T) {
	newApp := func() (*kingpin.Application, *kingpin.FlagClause) {
		app := kingpin.New("test", "").DefaultEnvars()
		flag := app.Flag("vm-reboot-timeout", "")
		flag.Default("120m").Duration()
		return app, flag
	}

	t.Run("not set falls back to default", func(t *testing.T) {
		app, flag := newApp()
		assert.False(t, isFlagSetByUser(app, flag, []string{}))
	})

	t.Run("set explicitly on the command line", func(t *testing.T) {
		app, flag := newApp()
		assert.True(t, isFlagSetByUser(app, flag, []string{"--vm-reboot-timeout=3h"}))
	})

	t.Run("a different flag being set does not count", func(t *testing.T) {
		app, flag := newApp()
		app.Flag("timeout", "").Default("1h").Duration()
		assert.False(t, isFlagSetByUser(app, flag, []string{"--timeout=2h"}))
	})

	t.Run("set via envar", func(t *testing.T) {
		app, flag := newApp()
		// DefaultEnvars derives the name as "<app.Name>_<flag.name>", uppercased.
		t.Setenv("TEST_VM_REBOOT_TIMEOUT", "3h")
		assert.True(t, isFlagSetByUser(app, flag, []string{}))
	})

	t.Run("unparseable args fail closed", func(t *testing.T) {
		app, flag := newApp()
		assert.False(t, isFlagSetByUser(app, flag, []string{"--does-not-exist"}))
	})
}
