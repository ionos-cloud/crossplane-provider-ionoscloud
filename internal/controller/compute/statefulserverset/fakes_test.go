package statefulserverset

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

const (
	create = "Create"
	ensure = "Ensure"
	update = "Update"
)

type fakeKubeLANController struct {
	v1alpha1.Lan
	v1alpha1.LanList
	error
}

type fakeKubeDataVolumeController struct {
	v1alpha1.Volume
	v1alpha1.VolumeList
	error
}

type fakeKubeServerSetController struct {
	methodCallCount map[string]int
}

func (f fakeKubeLANController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error) {
	return f.Lan, f.error
}

func (f fakeKubeLANController) Get(ctx context.Context, lanName, ns string) (*v1alpha1.Lan, error) {
	return &f.Lan, f.error
}

func (f fakeKubeLANController) Delete(ctx context.Context, name, namespace string) error {
	return f.error
}

func (f fakeKubeLANController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) error {
	return f.error
}

func (f fakeKubeLANController) ListLans(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.LanList, error) {
	return &f.LanList, f.error
}

func (f fakeKubeLANController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error) {
	return f.Lan, f.error
}

func (f fakeKubeDataVolumeController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error) {
	return f.Volume, f.error
}

func (f fakeKubeDataVolumeController) ListVolumes(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.VolumeList, error) {
	return &f.VolumeList, f.error
}

func (f fakeKubeDataVolumeController) Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error) {
	return &f.Volume, f.error
}

func (f fakeKubeDataVolumeController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error) {
	return f.Volume, f.error
}

func (f fakeKubeDataVolumeController) Delete(ctx context.Context, name, namespace string) error {
	return f.error
}

func (f fakeKubeDataVolumeController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, version int) error {
	return f.error
}

func (f *fakeKubeServerSetController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.ServerSet, error) {
	f.methodCallCount[create]++
	return nil, nil
}

func (f *fakeKubeServerSetController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, w WaitUntilAvailable) error {
	f.methodCallCount[ensure]++
	return nil
}

func (f *fakeKubeServerSetController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet) (v1alpha1.ServerSet, error) {
	f.methodCallCount[update]++
	return v1alpha1.ServerSet{}, nil

}

func fakeKubeClientWithObjs(objs ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func fakeKubeClientWithFunc(funcs interceptor.Funcs) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(funcs).Build()
}

func fakeWaitUntilAvailable(ctx context.Context, timeoutInMinutes time.Duration, fn kube.IsResourceReady, name, namespace string) error {
	return nil
}
