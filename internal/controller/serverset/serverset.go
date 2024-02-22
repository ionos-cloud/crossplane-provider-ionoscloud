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
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	ionoscloud "github.com/ionos-cloud/sdk-go/v6"
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

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	servers, err := c.getServersFromServerSet(ctx, cr.Spec.ForProvider.Template.Metadata.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.Replicas = len(servers)
	//we need to re-create servers. go to create
	if len(servers) < cr.Spec.ForProvider.Replicas {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	areServersUpToDate := areServersUpToDate(cr.Spec.ForProvider.Template.Spec, servers)
	//only update
	if areServersUpToDate == false {
		return managed.ExternalObservation{
			ResourceExists:    true,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
			Diff:              "servers are not up to date",
		}, nil
	}

	areNicsUpToDate := false
	//todo check nic parameters are same as template
	if areNicsUpToDate, err = c.areNicsUpToDate(ctx, cr); err != nil {
		return managed.ExternalObservation{}, err
	}
	if areNicsUpToDate == false {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

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
		if err := c.ensureBootVolume(ctx, cr, i); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureServer(ctx, cr, i); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureVolumeClaim(); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := c.ensureNICs(ctx, cr, i); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	// When all conditions are met, the managed resource is considered available
	cr.Status.SetConditions(xpv1.Available())
	meta.SetExternalName(cr, cr.Name)
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
	//how do we know if we want to update servers or nic params?
	update, err2 := c.updateServersFromTemplate(ctx, cr.Spec.ForProvider.Template)
	if err2 != nil {
		return update, err2
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) updateServersFromTemplate(ctx context.Context, templateSpec v1alpha1.ServerSetTemplate) (managed.ExternalUpdate, error) {
	servers, err := c.getServersFromServerSet(ctx, templateSpec.Metadata.Name)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	for _, serverObj := range servers {
		serverObj.Spec.ForProvider.RAM = templateSpec.Spec.RAM
		serverObj.Spec.ForProvider.Cores = templateSpec.Spec.Cores
		serverObj.Spec.ForProvider.CPUFamily = templateSpec.Spec.CPUFamily

		if err := c.kube.Update(ctx, &serverObj); err != nil {
			fmt.Printf("error updating server %w", err)
			return managed.ExternalUpdate{}, err
		}
	}
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	cr.SetConditions(xpv1.Deleting())

	fmt.Printf("Deleting: %+v", cr)

	if err := c.kube.DeleteAllOf(ctx, &v1alpha1.Nic{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	// delete all servers
	if err := c.kube.DeleteAllOf(ctx, &v1alpha1.Server{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	if err := c.kube.DeleteAllOf(ctx, &v1alpha1.Volume{}, client.InNamespace(cr.Namespace), client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return err
	}

	return nil
}

func (c *external) ensureBootVolume(ctx context.Context, cr *v1alpha1.ServerSet, idx int) error {
	c.log.Info("Ensuring BootVolume")
	name := getNameFromIndex(cr.Name, "volume", idx)
	ns := cr.Namespace
	volume, err := c.getVolume(ctx, name, ns)

	if err != nil {
		if apiErrors.IsNotFound(err) {
			return c.createBootVolume(ctx, cr, idx)
		}
		return err
	}
	fmt.Println("Server State: ", volume.Status.AtProvider.State)

	return nil
}

func (c *external) ensureVolumeClaim() error {
	c.log.Info("Ensuring Volume")

	return nil
}

func (c *external) ensureServer(ctx context.Context, cr *v1alpha1.ServerSet, idx int) error {
	c.log.Info("Ensuring Server")

	name := getNameFromIndex(cr.Name, "server", idx)
	ns := cr.Namespace
	obj, err := c.getServer(ctx, name, ns)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return c.createServer(ctx, cr, idx)
		}
		return err
	}

	fmt.Println("Server State: ", obj.Status.AtProvider.State)

	// check if the server is up and running
	fmt.Println("we have to check if the server is up and running")

	// check if the claims are mounted to the server
	fmt.Println("we have to check if the claims are mounted to the server")

	return nil
}

func (c *external) getServer(ctx context.Context, name, ns string) (*v1alpha1.Server, error) {
	obj := &v1alpha1.Server{}
	if err := c.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}, obj); err != nil {
		return nil, err
	}

	return obj, nil
}
func (c *external) getVolume(ctx context.Context, volumeName, ns string) (*v1alpha1.Volume, error) {
	obj := &v1alpha1.Volume{}
	if err := c.kube.Get(ctx, types.NamespacedName{
		Namespace: ns,
		Name:      volumeName,
	}, obj); err != nil {
		return nil, err
	}

	return obj, nil
}
func (c *external) createServer(ctx context.Context, cr *v1alpha1.ServerSet, idx int) error {
	c.log.Info("Creating Server")
	serverType := "server"
	serverObj := v1alpha1.Server{

		ObjectMeta: metav1.ObjectMeta{
			Name:      getNameFromIndex(cr.Name, serverType, idx),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.ServerSpec{
			ForProvider: v1alpha1.ServerParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             getNameFromIndex(cr.Name, serverType, idx),
				Cores:            cr.Spec.ForProvider.Template.Spec.Cores,
				RAM:              cr.Spec.ForProvider.Template.Spec.RAM,
				AvailabilityZone: "AUTO",
				CPUFamily:        cr.Spec.ForProvider.Template.Spec.CPUFamily,
				VolumeCfg: v1alpha1.VolumeConfig{
					VolumeIDRef: &xpv1.Reference{
						Name: getNameFromIndex(cr.Name, "volume", idx),
					},
					//	          selector:
					//            matchLabels:
					//              "cloud.ionos.com/volume-name": "volume-name"
					//              "cloud.ionos.com/replica_server_index": "0"
				},
			},
		}}
	serverObj.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := c.kube.Create(ctx, &serverObj); err != nil {
		fmt.Println("error creating server")
		fmt.Println(err.Error())
		return err
	}
	if err := WaitForResourceToBePopulated(ctx, 5*time.Minute, IsServerAvailable, c, getNameFromIndex(cr.Name, serverType, idx), cr.Namespace); err != nil {
		return fmt.Errorf("while waiting for server to be populated %w ", err)
	}
	return nil
}

// createBootVolume creates a volume CR and waits until in reaches AVAILABLE state
func (c *external) createBootVolume(ctx context.Context, cr *v1alpha1.ServerSet, idx int) error {
	c.log.Info("Creating Volume")
	resourceType := "volume"
	volumeObj := v1alpha1.Volume{

		ObjectMeta: metav1.ObjectMeta{
			Name:      getNameFromIndex(cr.Name, resourceType, idx),
			Namespace: cr.Namespace,
			Labels: map[string]string{
				serverSetLabel: cr.Name,
			},
		},
		ManagementPolicies: xpv1.ManagementPolicies{"*"},
		Spec: v1alpha1.VolumeSpec{
			ForProvider: v1alpha1.VolumeParameters{
				DatacenterCfg:    cr.Spec.ForProvider.DatacenterCfg,
				Name:             getNameFromIndex(cr.Name, resourceType, idx),
				AvailabilityZone: "AUTO",
				Size:             cr.Spec.ForProvider.BootVolumeTemplate.Spec.Size,
				Type:             cr.Spec.ForProvider.BootVolumeTemplate.Spec.Type,
				Image:            cr.Spec.ForProvider.BootVolumeTemplate.Spec.Image,
				//todo add to template(?)
				ImagePassword: "imagePassword776",
			},
		}}
	volumeObj.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
	if err := c.kube.Create(ctx, &volumeObj); err != nil {
		return err
	}
	if err := WaitForResourceToBePopulated(ctx, 5*time.Minute, IsVolumeAvailable, c, getNameFromIndex(cr.Name, resourceType, idx), cr.Namespace); err != nil {
		return err
	}

	return nil
}

func IsServerAvailable(ctx context.Context, c *external, name, namespace string) (bool, error) {
	kubeServer, err := c.getServer(ctx, name, namespace)
	if kubeServer != nil && kubeServer.Status.AtProvider.ServerID != "" && strings.EqualFold(kubeServer.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
	}
	return false, err
}

func IsVolumeAvailable(ctx context.Context, c *external, name, namespace string) (bool, error) {
	kubeVolume, err := c.getVolume(ctx, name, namespace)
	if kubeVolume != nil && kubeVolume.Status.AtProvider.VolumeID != "" && strings.EqualFold(kubeVolume.Status.AtProvider.State, ionoscloud.Available) {
		return true, nil
	}
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
	}
	return false, err
}

// IsResourcePopulatedFunc polls kube api to see if resource is available and observed(status populated)
type IsResourcePopulatedFunc func(ctx context.Context, c *external, name, namespace string) (bool, error)

// WaitForResourceToBePopulated - keeps retrying until resource available and observed(status populated), or until ctx is cancelled
func WaitForResourceToBePopulated(ctx context.Context, timeoutInMinutes time.Duration, fn IsResourcePopulatedFunc, c *external, name, namespace string) error {
	if c == nil {
		return fmt.Errorf("external client is nil")
	}
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	err := retry.RetryContext(ctx, timeoutInMinutes, func() *retry.RetryError {
		isDeleted, err := fn(ctx, c, name, namespace)
		if isDeleted {
			return nil
		}
		if err != nil {
			retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("resource with name %v found, still trying ", name))
	})
	return err
}
func (c *external) ensureNICs(ctx context.Context, cr *v1alpha1.ServerSet, idx int) error {
	c.log.Info("Ensuring NIC")

	srv, err := c.getServer(ctx, getNameFromIndex(cr.Name, "server", idx), cr.GetNamespace())
	if err != nil {
		return err
	}

	// check if the NIC is attached to the server
	fmt.Printf("we have to check if the NIC is attached to the server %s ", cr.Name)

	for nicx := range cr.Spec.ForProvider.Template.Spec.NICs {
		if err := c.ensureNIC(ctx, cr, srv.Status.AtProvider.ServerID, cr.Spec.ForProvider.Template.Spec.NICs[nicx].Reference, idx); err != nil {
			return err
		}
	}

	return nil
}

// areServersUpToDate checks if replicas and template params are equal to server obj params
func areServersUpToDate(templateParams v1alpha1.ServerSetTemplateSpec, servers []v1alpha1.Server) bool {

	for _, serverObj := range servers {
		if serverObj.Spec.ForProvider.Cores != templateParams.Cores {
			return false
		}
		if serverObj.Spec.ForProvider.RAM != templateParams.RAM {
			return false
		}
		if serverObj.Spec.ForProvider.CPUFamily != templateParams.CPUFamily {
			return false
		}
	}

	return true
}

// areNicsUpToDate gets nic k8s crs and checks if the correct number of NICs are created
func (c *external) areNicsUpToDate(ctx context.Context, cr *v1alpha1.ServerSet) (bool, error) {
	c.log.Info("Ensuring NIC")

	nicList := &v1alpha1.NicList{}
	if err := c.kube.List(ctx, nicList, client.MatchingLabels{
		serverSetLabel: cr.Name,
	}); err != nil {
		return false, err
	}

	if len(nicList.Items) != cr.Spec.ForProvider.Replicas {
		return false, nil
	}

	return true, nil
}

func (c *external) ensureNIC(ctx context.Context, cr *v1alpha1.ServerSet, serverID, lanName string, idx int) error {
	// get the network
	resourceType := "nic"
	nicName := getNameFromIndex(cr.Name, resourceType, idx)
	network := v1alpha1.Lan{}
	if err := c.kube.Get(ctx, types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      lanName,
	}, &network); err != nil {
		return err
	}

	lanID := network.Status.AtProvider.LanID
	observedNic := v1alpha1.Nic{}
	err := c.kube.Get(ctx, types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      nicName,
	}, &observedNic)
	if err != nil && !apiErrors.IsNotFound(err) {
		return err
	}
	// no NIC found, create one
	if apiErrors.IsNotFound(err) {
		c.log.Info("Creating NIC", "name", nicName)
		createNic := &v1alpha1.Nic{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nicName,
				Namespace: cr.GetNamespace(),
				Labels: map[string]string{
					serverSetLabel: cr.Name,
				},
			},
			ManagementPolicies: xpv1.ManagementPolicies{"*"},
			Spec: v1alpha1.NicSpec{
				ForProvider: v1alpha1.NicParameters{
					Name:          nicName,
					DatacenterCfg: cr.Spec.ForProvider.DatacenterCfg,
					ServerCfg: v1alpha1.ServerConfig{
						ServerID: serverID,
					},
					LanCfg: v1alpha1.LanConfig{
						LanID: lanID,
					},
				},
			},
		}
		createNic.SetProviderConfigReference(cr.Spec.ProviderConfigReference)
		return c.kube.Create(ctx, createNic)
	}

	// NIC found, check if it's attached to the server
	if !strings.EqualFold(observedNic.Status.AtProvider.State, ionoscloud.Available) {
		return fmt.Errorf("observedNic %s got state %s but expected %s", observedNic.GetName(), observedNic.Status.AtProvider.State, ionoscloud.Available)
	}

	// check if we have to update the NIC

	return nil
}
func (c *external) getServersFromServerSet(ctx context.Context, name string) ([]v1alpha1.Server, error) {
	serverList := &v1alpha1.ServerList{}
	if err := c.kube.List(ctx, serverList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return serverList.Items, nil
}

// getNameFromIndex - generates name consisting of name, kind and index
func getNameFromIndex(resourceName, resourceType string, idx int) string {
	return fmt.Sprintf("%s-%s-%d", resourceName, resourceType, idx)
}
