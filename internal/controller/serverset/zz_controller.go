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

package serverset

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/compute/server"
)

const (
	errUnexpectedObject = "managed resource is not an Volume resource"

	errTrackPCUsage = "cannot track ProviderConfig usage"

	serverSetLabel = "ionoscloud.com/serverset"
)

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return nil, errors.New(errUnexpectedObject)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	svc, err := clients.ConnectForCRD(ctx, mg, c.kube, c.usage)
	if err != nil {
		return nil, err
	}

	return &external{
		kube:    c.kube,
		service: &server.APIClient{IonosServices: svc},
		log:     c.log,
	}, err
}

// external observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.

	service server.Client
	log     logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	serverList := &v1alpha1.ServerList{}
	if err := c.kube.List(ctx, serverList, client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return managed.ExternalObservation{}, err
	}

	fmt.Printf("Got a total of %d servers", len(serverList.Items))
	cr.Status.AtProvider.Replicas = len(serverList.Items)

	// ensure we have cr.Spec.Replicas number of servers
	if len(serverList.Items) != cr.Spec.ForProvider.Replicas {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	// TODO: check for volume claims

	// TODO: check for NICs attached to the servers

	return managed.ExternalObservation{
		// Return false when the externalServerSet resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the externalServerSet resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the externalServerSet
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	cr.Status.SetConditions(xpv1.Creating())

	// for n times of cr.Spec.Replicas, create a server
	// for each server, create a volume
	c.log.Info("Creating a new ServerSet", "replicas", cr.Spec.ForProvider.Replicas)

	for i := 0; i < cr.Spec.ForProvider.Replicas; i++ {
		c.log.Info("Creating a new Server", "index", i)
		if err := c.ensureBootVolumeClaim(); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureServer(ctx, cr, i); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureVolumeClaim(); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureNIC(); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	// When all conditions are met, the managed resource is considered available
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	cr.SetConditions(xpv1.Deleting())

	fmt.Printf("Deleting: %+v", cr)

	if err := c.kube.DeleteAllOf(ctx, &v1alpha1.Server{}, client.InNamespace(cr.Namespace)); err != nil {
		return err
	}

	return nil
}

func (c *external) ensureBootVolumeClaim() error {
	c.log.Info("Ensuring VolumeClaim")

	return nil
}

func (c *external) ensureVolumeClaim() error {
	c.log.Info("Ensuring Volume")

	return nil
}

func (c *external) ensureServer(ctx context.Context, cr *v1alpha1.ServerSet, idx int) error {
	c.log.Info("Ensuring Server")

	name := fmt.Sprintf("%s-%d", cr.Spec.ForProvider.Template.Metadata.Name, idx)
	ns := cr.Namespace

	obj := &v1alpha1.Server{}
	if err := c.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj); err != nil {
		if apiErrors.IsNotFound(err) {
			return c.createServer(ctx, cr, idx)
		}
		return err
	}
	// can be "AVAILABLE"
	if obj.Status.AtProvider.State == "AVAILABLE" {
		return nil
	}

	fmt.Println("Server State: ", obj.Status.AtProvider.State)

	// check if the server is up and running
	fmt.Println("we have to check if the server is up and running")

	// check if the claims are mounted to the server
	fmt.Println("we have to check if the claims are mounted to the server")

	return nil
}

func (c *external) createServer(ctx context.Context, cr *v1alpha1.ServerSet, idx int) error {
	c.log.Info("Creating Server")
	fmt.Println("Creating Server")

	if err := c.kube.Create(ctx, &v1alpha1.Server{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", cr.Spec.ForProvider.Template.Metadata.Name, idx),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.ServerSpec{
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             fmt.Sprintf("%s-%d", cr.Spec.ForProvider.Template.Metadata.Name, idx),
				Cores:            cr.Spec.ForProvider.Template.Spec.Cores,
				RAM:              cr.Spec.ForProvider.Template.Spec.RAM,
				AvailabilityZone: "AUTO",
				CPUFamily:        cr.Spec.ForProvider.Template.Spec.CPUFamily,
			},
		},
	}); err != nil {
		fmt.Println("error creating server")
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func (c *external) ensureNIC() error {
	c.log.Info("Ensuring NIC")

	return nil
}
