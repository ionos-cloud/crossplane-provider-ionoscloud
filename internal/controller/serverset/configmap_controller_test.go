package serverset

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

// testSubstServerSetName is an arbitrary ServerSet name used across this file's fixtures.
const testSubstServerSetName = "sset1"

// Test_kubeConfigmapController_ConcurrentAccess guards against concurrent map read/write on the
// substConfigMap shared across all ServerSets. This repo's CI (make test, via the vendored
// build/makelib/golang.mk) runs with CGO_ENABLED=0 and no -race, so this test relies on Go's
// runtime detecting concurrent map writes unconditionally (not just under -race) - which is what
// actually crashed in production. For richer diagnostics on a local run, `-race` needs
// CGO_ENABLED=1 explicitly, e.g. `CGO_ENABLED=1 go test -race ./internal/controller/serverset/...`.
func Test_kubeConfigmapController_ConcurrentAccess(t *testing.T) {
	k := &kubeConfigmapController{log: logging.NewNopLogger()}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("serverset-%d", i)
			k.SetSubstitutionConfigMap(name, stateMapNamespace)
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
// pattern as kubeConfigmapController.substConfigMap above). See the -race caveat on
// Test_kubeConfigmapController_ConcurrentAccess above - this repo's CI can't run with -race.
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

func newTestConfigmapController(objs ...client.Object) *kubeConfigmapController {
	return &kubeConfigmapController{
		kube: fakeKubeClientObjs(objs...),
		log:  logging.NewNopLogger(),
	}
}

func Test_kubeConfigmapController_FetchSubstitutionFromMap(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "subst-cm", Namespace: stateMapNamespace},
		Data:       map[string]string{"0.0.key": "value"},
	}

	t.Run("configmap exists: returns the value", func(t *testing.T) {
		k := newTestConfigmapController(cm)
		k.SetSubstitutionConfigMap(testSubstServerSetName, stateMapNamespace)
		k.substConfigMap[testSubstServerSetName].name = "subst-cm"

		got := k.FetchSubstitutionFromMap(context.Background(), testSubstServerSetName, "key", 0, 0)
		assert.Equal(t, "value", got)
	})

	t.Run("configmap missing: returns empty string", func(t *testing.T) {
		k := newTestConfigmapController()
		k.SetSubstitutionConfigMap(testSubstServerSetName, stateMapNamespace)
		k.substConfigMap[testSubstServerSetName].name = "does-not-exist"

		got := k.FetchSubstitutionFromMap(context.Background(), testSubstServerSetName, "key", 0, 0)
		assert.Empty(t, got)
	})
}

func Test_kubeConfigmapController_CreateOrUpdate(t *testing.T) {
	cr := &v1alpha1.ServerSet{ObjectMeta: metav1.ObjectMeta{Name: testSubstServerSetName}}

	t.Run("configmap doesn't exist: creates it", func(t *testing.T) {
		k := newTestConfigmapController()
		k.SetSubstitutionConfigMap(testSubstServerSetName, stateMapNamespace)
		k.SetIdentity(testSubstServerSetName, "0.0.key", "value")

		require.NoError(t, k.CreateOrUpdate(context.Background(), cr))

		got, err := k.Get(context.Background(), testSubstServerSetName, stateMapNamespace)
		require.NoError(t, err)
		assert.Equal(t, "value", got.Data["0.0.key"])
	})

	t.Run("configmap exists with different data: updates it", func(t *testing.T) {
		existing := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testSubstServerSetName, Namespace: stateMapNamespace},
			Data:       map[string]string{"0.0.key": "old"},
		}
		k := newTestConfigmapController(existing)
		k.SetSubstitutionConfigMap(testSubstServerSetName, stateMapNamespace)
		k.SetIdentity(testSubstServerSetName, "0.0.key", "new")

		require.NoError(t, k.CreateOrUpdate(context.Background(), cr))

		got, err := k.Get(context.Background(), testSubstServerSetName, stateMapNamespace)
		require.NoError(t, err)
		assert.Equal(t, "new", got.Data["0.0.key"])
	})

	t.Run("configmap exists with same data: no-op", func(t *testing.T) {
		existing := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testSubstServerSetName, Namespace: stateMapNamespace},
			Data:       map[string]string{"0.0.key": "value"},
		}
		k := newTestConfigmapController(existing)
		k.SetSubstitutionConfigMap(testSubstServerSetName, stateMapNamespace)
		k.SetIdentity(testSubstServerSetName, "0.0.key", "value")

		assert.NoError(t, k.CreateOrUpdate(context.Background(), cr))
	})
}

func Test_kubeConfigmapController_Delete(t *testing.T) {
	cm := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: testSubstServerSetName, Namespace: stateMapNamespace}}
	k := newTestConfigmapController(cm)
	k.SetSubstitutionConfigMap(testSubstServerSetName, stateMapNamespace)

	require.NoError(t, k.Delete(context.Background(), testSubstServerSetName))

	_, err := k.Get(context.Background(), testSubstServerSetName, stateMapNamespace)
	assert.Error(t, err, "configmap must be gone after Delete")
}

func Test_kubeConfigmapController_isDeleted(t *testing.T) {
	t.Run("not found: clears the map entry and reports deleted", func(t *testing.T) {
		k := newTestConfigmapController()
		k.SetSubstitutionConfigMap(testSubstServerSetName, stateMapNamespace)

		// No ConfigMap matching testSubstServerSetName exists in the fake client, so Get returns
		// NotFound and isDeleted must clear the map entry keyed by that same name.
		deleted, err := k.isDeleted(context.Background(), testSubstServerSetName, stateMapNamespace)
		require.NoError(t, err)
		assert.True(t, deleted)
		assert.Nil(t, k.substConfigMap[testSubstServerSetName])
	})

	t.Run("still present: reports not deleted", func(t *testing.T) {
		cm := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: testSubstServerSetName, Namespace: stateMapNamespace}}
		k := newTestConfigmapController(cm)

		deleted, err := k.isDeleted(context.Background(), testSubstServerSetName, stateMapNamespace)
		require.NoError(t, err)
		assert.False(t, deleted)
	})
}
