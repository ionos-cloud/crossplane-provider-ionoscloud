/*
Copyright 2020 The Crossplane Authors.

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

package postgres

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	ionoscloud "github.com/ionos-cloud/sdk-go-dbaas-postgres"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/dbaas/postgres/v1alpha1"
	apisv1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/v1alpha1"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/clients/dbaas/postgres"
)

const (
	errNotCluster   = "managed resource is not a Cluster custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A NoOpService does nothing.
type NoOpService struct{}

// var (
//	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
// )

// Setup adds a controller that reconciles Cluster managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(v1alpha1.ClusterGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ClusterGroupVersionKind),
		managed.WithExternalConnecter(&connectorCluster{
			kube:  mgr.GetClient(),
			usage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			log:   l}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1alpha1.Cluster{}).
		Complete(r)
}

// A connectorCluster is expected to produce an ExternalClient when its Connect method
// is called.
type connectorCluster struct {
	kube  client.Client
	usage resource.Tracker
	log   logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connectorCluster) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return nil, errors.New(errNotCluster)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := clients.NewIonosClients(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &externalCluster{service: &postgres.ClusterAPIClient{IonosServices: svc}, log: c.log}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// externalCluster resource to ensure it reflects the managed resource's desired state.
type externalCluster struct {
	// A 'client' used to connect to the externalCluster resource API. In practice this
	// would be something like an AWS SDK client.
	service postgres.ClusterClient
	log     logging.Logger
}

func (c *externalCluster) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCluster)
	}
	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, nil
	}

	cr.Status.AtProvider.ClusterID = id
	cluster, resp, err := c.service.GetCluster(ctx, id)

	if err != nil {
		retErr := fmt.Errorf("failed to get cluster by id. Request: %v: %w", resp.RequestURL, err)
		if resp.StatusCode == http.StatusNotFound {
			retErr = nil
		}
		return managed.ExternalObservation{
			ResourceExists:    false,
			ResourceUpToDate:  false,
			ConnectionDetails: managed.ConnectionDetails{},
		}, retErr
	}

	cr.Status.AtProvider.State = string(*cluster.Metadata.State)
	c.log.Debug(fmt.Sprintf("Observing state %v...", cr.Status.AtProvider.State))
	setReadyCondition(cr.Status.AtProvider.State, cr)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isClusterUpToDate(cr, cluster),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func setReadyCondition(apiState string, mg resource.Managed) {
	switch apiState {
	case string(ionoscloud.AVAILABLE):
		mg.SetConditions(xpv1.Available())
	case string(ionoscloud.DESTROYING):
		mg.SetConditions(xpv1.Deleting())
	case string(ionoscloud.FAILED):
		mg.SetConditions(xpv1.Unavailable())
	case string(ionoscloud.BUSY):
		mg.SetConditions(xpv1.Creating())
	default:
		mg.SetConditions(xpv1.Unavailable())
	}
}

func isClusterUpToDate(cr *v1alpha1.Cluster, clusterResponse ionoscloud.ClusterResponse) bool {
	switch {
	case cr == nil && clusterResponse.Properties == nil:
		return true
	case cr == nil && clusterResponse.Properties != nil:
		return false
	case cr != nil && clusterResponse.Properties == nil:
		return false
	}

	if *clusterResponse.Metadata.State == ionoscloud.BUSY {
		return true
	}

	if strings.Compare(cr.Spec.ForProvider.DisplayName, *clusterResponse.Properties.DisplayName) != 0 {
		return false
	}
	return true
}

func (c *externalCluster) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCluster)
	}

	cluster := ionoscloud.CreateClusterRequest{
		Properties: &ionoscloud.CreateClusterProperties{
			PostgresVersion:     &cr.Spec.ForProvider.PostgresVersion,
			Instances:           &cr.Spec.ForProvider.Instances,
			Cores:               &cr.Spec.ForProvider.Cores,
			Ram:                 &cr.Spec.ForProvider.RAM,
			StorageSize:         &cr.Spec.ForProvider.StorageSize,
			StorageType:         (*ionoscloud.StorageType)(&cr.Spec.ForProvider.StorageType),
			Connections:         clusterConnections(cr.Spec.ForProvider.Connections),
			Location:            (*ionoscloud.Location)(&cr.Spec.ForProvider.Location),
			DisplayName:         &cr.Spec.ForProvider.DisplayName,
			Credentials:         clusterCredentials(cr.Spec.ForProvider.Credentials),
			SynchronizationMode: (*ionoscloud.SynchronizationMode)(&cr.Spec.ForProvider.SynchronizationMode),
		},
	}
	fromBackup, err := clusterFromBackup(cr.Spec.ForProvider.FromBackup)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	if fromBackup != nil {
		cluster.Properties.SetFromBackup(*fromBackup)
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		cluster.Properties.SetMaintenanceWindow(*window)
	}

	created, apiResponse, err := c.service.PostCluster(ctx, cluster)
	creation := managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}
	if err != nil {
		return creation, fmt.Errorf("failed to create Cluster: %w, apiResponse: %v", err, apiResponse.Status)
	}

	cr.Status.AtProvider.ClusterID = *created.Id
	meta.SetExternalName(cr, *created.Id)
	creation.ExternalNameAssigned = true
	c.log.Debug(fmt.Sprintf("External name: %v", meta.GetExternalName(cr)))
	return creation, nil
}

func clusterConnections(connections []v1alpha1.Connection) *[]ionoscloud.Connection {
	connects := make([]ionoscloud.Connection, 0)
	for _, connection := range connections {
		datacenterID := connection.DatacenterID
		lanID := connection.LanID
		cidr := connection.Cidr
		connects = append(connects, ionoscloud.Connection{
			DatacenterId: &datacenterID,
			LanId:        &lanID,
			Cidr:         &cidr,
		})
	}
	return &connects
}

func clusterMaintenanceWindow(window v1alpha1.MaintenanceWindow) *ionoscloud.MaintenanceWindow {
	if window.Time != "" && window.DayOfTheWeek != "" {
		return &ionoscloud.MaintenanceWindow{
			Time:         &window.Time,
			DayOfTheWeek: (*ionoscloud.DayOfTheWeek)(&window.DayOfTheWeek),
		}
	}
	return nil
}

func clusterCredentials(creds v1alpha1.DBUser) *ionoscloud.DBUser {
	return &ionoscloud.DBUser{
		Username: &creds.Username,
		Password: &creds.Password,
	}
}

func clusterFromBackup(req v1alpha1.CreateRestoreRequest) (*ionoscloud.CreateRestoreRequest, error) {
	if req.BackupID != "" && req.RecoveryTargetTime != "" {
		recoveryTime, err := time.Parse(time.RFC3339, req.RecoveryTargetTime)
		if err != nil {
			return nil, err
		}
		return &ionoscloud.CreateRestoreRequest{
			BackupId:           &req.BackupID,
			RecoveryTargetTime: &ionoscloud.IonosTime{Time: recoveryTime},
		}, nil
	}
	return nil, nil
}

func (c *externalCluster) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCluster)
	}

	clusterID := cr.Status.AtProvider.ClusterID
	cluster := ionoscloud.PatchClusterRequest{
		Properties: &ionoscloud.PatchClusterProperties{
			PostgresVersion: &cr.Spec.ForProvider.PostgresVersion,
			Instances:       &cr.Spec.ForProvider.Instances,
			Cores:           &cr.Spec.ForProvider.Cores,
			Ram:             &cr.Spec.ForProvider.RAM,
			StorageSize:     &cr.Spec.ForProvider.StorageSize,
			Connections:     clusterConnections(cr.Spec.ForProvider.Connections),
			DisplayName:     &cr.Spec.ForProvider.DisplayName,
		},
	}
	if window := clusterMaintenanceWindow(cr.Spec.ForProvider.MaintenanceWindow); window != nil {
		cluster.Properties.SetMaintenanceWindow(*window)
	}

	_, apiResponse, err := c.service.PatchCluster(ctx, clusterID, cluster)
	update := managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}
	if err != nil {
		return update, fmt.Errorf("failed to update Cluster: %w, apiResponse: %v", err, apiResponse.Status)
	}
	return update, nil
}

func (c *externalCluster) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.Cluster)
	if !ok {
		return errors.New(errNotCluster)
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return nil
	}

	cluster, _, err := c.service.GetCluster(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster state")
	}

	if *cluster.Metadata.State == ionoscloud.DESTROYING {
		return nil
	}

	err = c.service.DeleteCluster(ctx, id)
	return errors.Wrap(err, "failed to delete cluster")
}
