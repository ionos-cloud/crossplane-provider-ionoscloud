package statefulserverset

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
)

const (
	create  = "Create"
	ensure  = "Ensure"
	update  = "Update"
	ddelete = "Delete"
)

type fakeKubeLANController struct {
	Lan     v1alpha1.Lan
	LanList v1alpha1.LanList
	Err     error
}

type fakeKubeDataVolumeController struct {
	Volume     v1alpha1.Volume
	VolumeList v1alpha1.VolumeList
	Err        error
}

type fakeKubeVolumeSelectorController struct {
	Volume v1alpha1.Volumeselector
	Err    error
}

type fakeKubeServerSetController struct {
	methodCallCount map[string]int
}

func (f fakeKubeLANController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error) {
	return f.Lan, f.Err
}

func (f fakeKubeLANController) Get(ctx context.Context, lanName, ns string) (*v1alpha1.Lan, error) {
	return &f.Lan, f.Err
}

func (f fakeKubeLANController) Delete(ctx context.Context, name, namespace string) error {
	return f.Err
}

func (f fakeKubeLANController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) error {
	return f.Err
}

func (f fakeKubeLANController) ListLans(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.LanList, error) {
	return &f.LanList, f.Err
}

func (f fakeKubeLANController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, lanIndex int) (v1alpha1.Lan, error) {
	return f.Lan, f.Err
}

func (f fakeKubeDataVolumeController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error) {
	return f.Volume, f.Err
}

func (f fakeKubeDataVolumeController) ListVolumes(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.VolumeList, error) {
	return &f.VolumeList, f.Err
}

func (f fakeKubeDataVolumeController) Get(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error) {
	return &f.Volume, f.Err
}

func (f fakeKubeDataVolumeController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, volumeIndex int) (v1alpha1.Volume, error) {
	return f.Volume, f.Err
}

func (f fakeKubeDataVolumeController) Delete(ctx context.Context, name, namespace string) error {
	return f.Err
}

func (f fakeKubeDataVolumeController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex, version int) error {
	return f.Err
}

func (f *fakeKubeServerSetController) Get(ctx context.Context, ssetName, ns string) (*v1alpha1.ServerSet, error) {
	return nil, nil
}

func (f *fakeKubeServerSetController) Create(ctx context.Context, cr *v1alpha1.StatefulServerSet) (*v1alpha1.ServerSet, error) {
	f.methodCallCount[create]++
	return nil, nil
}
func (f *fakeKubeServerSetController) Delete(ctx context.Context, name, namespace string) error {
	f.methodCallCount[ddelete]++
	return nil
}
func (f *fakeKubeServerSetController) Ensure(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	f.methodCallCount[ensure]++
	return nil
}

func (f *fakeKubeServerSetController) Update(ctx context.Context, cr *v1alpha1.StatefulServerSet, forceUpdate bool) (v1alpha1.ServerSet, error) {
	f.methodCallCount[update]++
	return v1alpha1.ServerSet{}, nil

}

func (f fakeKubeVolumeSelectorController) CreateOrUpdate(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	return f.Err
}

func (f fakeKubeVolumeSelectorController) Get(ctx context.Context, name, ns string) (*v1alpha1.Volumeselector, error) {
	return &f.Volume, f.Err
}

func (f fakeKubeVolumeSelectorController) IsAvailable(ctx context.Context, name, ns string) (bool, error) {
	return true, f.Err
}

func (f fakeKubeVolumeSelectorController) Delete(ctx context.Context, name, ns string) error {
	return nil
}
func fakeKubeClientWithObjs(objs ...client.Object) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func fakeKubeClientWithSSetRelatedObjects() client.WithWatch {
	return fakeKubeClientWithObjs(
		createSSet(), createServer1(), createServer2(),
		createBootVolume1(), createBootVolume2(),
		createNIC1(), createNIC2(),
	)
}

func fakeKubeClientWithFunc(funcs interceptor.Funcs) client.WithWatch {
	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)       // Add the core k8s types to the Scheme
	v1alpha1.AddToScheme(scheme) // Add our custom types from v1alpha to the Scheme
	return fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(funcs).Build()
}
