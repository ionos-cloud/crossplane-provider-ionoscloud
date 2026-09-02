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

func newTestConfigmapController(objs ...client.Object) *kubeConfigmapController {
	return &kubeConfigmapController{
		kube: fakeKubeClientObjs(objs...),
		log:  logging.NewNopLogger(),
	}
}

func Test_kubeConfigmapController_FetchSubstitutionFromMap(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "subst-cm", Namespace: "default"},
		Data:       map[string]string{"0.0.key": "value"},
	}

	t.Run("configmap exists: returns the value", func(t *testing.T) {
		k := newTestConfigmapController(cm)
		k.SetSubstitutionConfigMap("sset1", "default")
		k.substConfigMap["sset1"].name = "subst-cm"

		got := k.FetchSubstitutionFromMap(context.Background(), "sset1", "key", 0, 0)
		assert.Equal(t, "value", got)
	})

	t.Run("configmap missing: returns empty string", func(t *testing.T) {
		k := newTestConfigmapController()
		k.SetSubstitutionConfigMap("sset1", "default")
		k.substConfigMap["sset1"].name = "does-not-exist"

		got := k.FetchSubstitutionFromMap(context.Background(), "sset1", "key", 0, 0)
		assert.Empty(t, got)
	})
}

func Test_kubeConfigmapController_CreateOrUpdate(t *testing.T) {
	cr := &v1alpha1.ServerSet{ObjectMeta: metav1.ObjectMeta{Name: "sset1"}}

	t.Run("configmap doesn't exist: creates it", func(t *testing.T) {
		k := newTestConfigmapController()
		k.SetSubstitutionConfigMap("sset1", "default")
		k.SetIdentity("sset1", "0.0.key", "value")

		require.NoError(t, k.CreateOrUpdate(context.Background(), cr))

		got, err := k.Get(context.Background(), "sset1", "default")
		require.NoError(t, err)
		assert.Equal(t, "value", got.Data["0.0.key"])
	})

	t.Run("configmap exists with different data: updates it", func(t *testing.T) {
		existing := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "sset1", Namespace: "default"},
			Data:       map[string]string{"0.0.key": "old"},
		}
		k := newTestConfigmapController(existing)
		k.SetSubstitutionConfigMap("sset1", "default")
		k.SetIdentity("sset1", "0.0.key", "new")

		require.NoError(t, k.CreateOrUpdate(context.Background(), cr))

		got, err := k.Get(context.Background(), "sset1", "default")
		require.NoError(t, err)
		assert.Equal(t, "new", got.Data["0.0.key"])
	})

	t.Run("configmap exists with same data: no-op", func(t *testing.T) {
		existing := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "sset1", Namespace: "default"},
			Data:       map[string]string{"0.0.key": "value"},
		}
		k := newTestConfigmapController(existing)
		k.SetSubstitutionConfigMap("sset1", "default")
		k.SetIdentity("sset1", "0.0.key", "value")

		assert.NoError(t, k.CreateOrUpdate(context.Background(), cr))
	})
}

func Test_kubeConfigmapController_Delete(t *testing.T) {
	cm := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "sset1", Namespace: "default"}}
	k := newTestConfigmapController(cm)
	k.SetSubstitutionConfigMap("sset1", "default")

	require.NoError(t, k.Delete(context.Background(), "sset1"))

	_, err := k.Get(context.Background(), "sset1", "default")
	assert.Error(t, err, "configmap must be gone after Delete")
}

func Test_kubeConfigmapController_isDeleted(t *testing.T) {
	t.Run("not found: clears the map entry and reports deleted", func(t *testing.T) {
		k := newTestConfigmapController()
		k.SetSubstitutionConfigMap("sset1", "default")

		// No ConfigMap named "sset1" exists in the fake client, so Get returns NotFound and
		// isDeleted must clear the map entry keyed by that same name.
		deleted, err := k.isDeleted(context.Background(), "sset1", "default")
		require.NoError(t, err)
		assert.True(t, deleted)
		assert.Nil(t, k.substConfigMap["sset1"])
	})

	t.Run("still present: reports not deleted", func(t *testing.T) {
		cm := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "sset1", Namespace: "default"}}
		k := newTestConfigmapController(cm)

		deleted, err := k.isDeleted(context.Background(), "sset1", "default")
		require.NoError(t, err)
		assert.False(t, deleted)
	})
}
