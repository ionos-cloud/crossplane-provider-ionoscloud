package serverset

import (
	"fmt"
	"sync"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
)

// Test_kubeConfigmapController_ConcurrentAccess guards against concurrent map read/write on the
// substConfigMap shared across all ServerSets. Run with `go test -race` to catch the data race,
// not just an occasional fatal error.
func Test_kubeConfigmapController_ConcurrentAccess(t *testing.T) {
	k := &kubeConfigmapController{log: logging.NewNopLogger()}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("serverset-%d", i)
			k.SetSubstitutionConfigMap(name, "default")
			k.SetIdentity(name, "0.0.key", "value")
		}(i)
	}
	wg.Wait()

	if got := len(k.substConfigMap); got != goroutines {
		t.Errorf("substConfigMap has %d entries, want %d", got, goroutines)
	}
}

// Test_getOrInitGlobalState_ConcurrentAccess is a regression test covering the sibling data race
// on the package-level globalStateMap (same "shared across concurrently-reconciled ServerSets"
// pattern as kubeConfigmapController.substConfigMap above). Run with `go test -race`.
func Test_getOrInitGlobalState_ConcurrentAccess(t *testing.T) {
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("serverset-global-%d", i)
			state := getOrInitGlobalState(name)
			state.Set("identifier", "key", "value")
		}(i)
	}
	wg.Wait()

	globalStateMapMu.Lock()
	defer globalStateMapMu.Unlock()
	for i := 0; i < goroutines; i++ {
		name := fmt.Sprintf("serverset-global-%d", i)
		if _, ok := globalStateMap[name]; !ok {
			t.Errorf("globalStateMap missing entry for %q", name)
		}
	}
}
