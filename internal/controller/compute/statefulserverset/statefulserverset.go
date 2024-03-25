/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package statefulserverset

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/types"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
)

const (
	errNotStatefulServerSet = "managed resource is not a StatefulServerSet custom resource"
	errTrackPCUsage         = "cannot track ProviderConfig usage"
)

const statefulServerSetLabel = "ionoscloud.com/statefulServerSet"

// A NoOpService does nothing.
type NoOpService struct{}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                 client.Client
	usage                resource.Tracker
	log                  logging.Logger
	dataVolumeController kubeDataVolumeControlManager
	sSetController       kubeSSetControlManager
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return nil, errors.New(errNotStatefulServerSet)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	if err != nil {
		return nil, err
	}
	return &external{
		kube:                 c.kube,
		service:              &server.APIClient{IonosServices: svc},
		dataVolumeController: c.dataVolumeController,
		sSetController:       c.sSetController,
		log:                  c.log,
	}, err
}

// external observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube                 client.Client
	service              server.Client
	dataVolumeController kubeDataVolumeControlManager
	sSetController       kubeSSetControlManager
	log                  logging.Logger
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotStatefulServerSet)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	serverSet := &v1alpha1.ServerSet{}
	nsName := computeSSetNsName(cr)
	if err := e.kube.Get(ctx, nsName, serverSet); err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		// Return false when the externalStatefulServerSet resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the externalStatefulServerSet resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the externalStatefulServerSet
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotStatefulServerSet)
	}

	cr.Status.SetConditions(xpv1.Creating())

	e.log.Info("Creating a new ServerSet", "replicas", cr.Spec.ForProvider.Replicas)
	for replicaIndex := 0; replicaIndex < cr.Spec.ForProvider.Replicas; replicaIndex++ {
		e.log.Info("Creating the data volumes")
		err := e.ensureDataVolumes(ctx, cr, replicaIndex)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("while ensuring data volumes %w", err)
		}
		e.log.Info("Creating the lans")
	}

	if err := e.ensureSSet(ctx, cr); err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("while ensuring server set %w", err)
	}

	// When all conditions are met, the managed resource is considered available
	meta.SetExternalName(cr, cr.Name)
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// externalStatefulServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v1alpha1.StatefulServerSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotStatefulServerSet)
	}

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalStatefulServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.StatefulServerSet)
	cr.SetConditions(xpv1.Deleting())
	if !ok {
		return errors.New(errNotStatefulServerSet)
	}
	if err := e.kube.DeleteAllOf(ctx, &v1alpha1.Volume{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		statefulServerSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	return nil
}

func (e *external) ensureDataVolumes(ctx context.Context, cr *v1alpha1.StatefulServerSet, replicaIndex int) error {
	e.log.Info("Ensuring the data volumes")
	for volumeIndex := range cr.Spec.ForProvider.Volumes {
		err := e.dataVolumeController.Ensure(ctx, cr, replicaIndex, volumeIndex)
		if err != nil {
			return err
		}
	}
	return nil
}
func (e *external) ensureSSet(ctx context.Context, cr *v1alpha1.StatefulServerSet) error {
	e.log.Info("Ensuring the server set")
	err := e.sSetController.Ensure(ctx, cr)
	if err != nil {
		return err
	}
	return nil
}

// generateNameFrom - generates name consisting of name, kind, index and version/second index
func generateNameFrom(resourceName, resourceType string, idx, version int) string {
	return fmt.Sprintf("%s-%s-%d-%d", resourceName, resourceType, idx, version)
}

func computeSSetNsName(cr *v1alpha1.StatefulServerSet) types.NamespacedName {
	ssName := cr.Name + "-" + cr.Spec.ForProvider.Template.Metadata.Name

	namespace := "default"
	if cr.Namespace != "" {
		namespace = cr.Namespace
	}

	return types.NamespacedName{
		Name:      ssName,
		Namespace: namespace,
	}
}
