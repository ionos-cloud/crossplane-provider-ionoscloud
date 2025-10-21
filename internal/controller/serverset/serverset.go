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
	"strconv"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	ionoscloud "github.com/ionos-cloud/sdk-go/v6"

	"github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	control "github.com/ionos-cloud/crossplane-provider-ionoscloud/internal/controller/compute/server"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/ccpatch/substitution"
	"github.com/ionos-cloud/crossplane-provider-ionoscloud/pkg/kube"
)

const (
	errUnexpectedObject = "managed resource is not an Volume resource"
	errTrackPCUsage     = "cannot track ProviderConfig usage"
)

const (
	// indexLabel is the label used to identify the server set by index
	indexLabel = "%s-%s-ri"

	nicIndexLabel = "%s-%s-ni"
	// versionLabel is the label used to identify the server set by version
	versionLabel = "%s-%s-v"
	// serverSetLabel is the label used to identify the server set resources. All resources created by a server set will have this label
	serverSetLabel = "serverset"
)

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube                    client.Client
	bootVolumeController    kubeBootVolumeControlManager
	nicController           kubeNicControlManager
	serverController        kubeServerControlManager
	firewallRuleController  kubeFirewallRuleControlManager
	kubeConfigmapController kubeConfigmapControlManager
	usage                   resource.Tracker
	log                     logging.Logger
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
	var err error
	if err = c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	return &external{
		kube:                   c.kube,
		log:                    c.log,
		bootVolumeController:   c.bootVolumeController,
		nicController:          c.nicController,
		serverController:       c.serverController,
		firewallRuleController: c.firewallRuleController,
		configMapController:    c.kubeConfigmapController,
	}, err
}

// external observes, then either creates, updates, or deletes an
// externalServerSet resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	// A 'client' used to connect to the externalServer resource API. In practice this
	// would be something like an IONOS Cloud SDK client.
	bootVolumeController   kubeBootVolumeControlManager
	nicController          kubeNicControlManager
	serverController       kubeServerControlManager
	firewallRuleController kubeFirewallRuleControlManager
	configMapController    kubeConfigmapControlManager
	log                    logging.Logger
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) { // nolint:gocyclo
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{}, nil
	}

	if meta.WasDeleted(cr) {
		return managed.ExternalObservation{}, nil
	}

	servers, err := GetServersOfSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	e.populateReplicasStatuses(ctx, cr, servers)

	// When a state map is configured in the sset, we need to retrieve it to check the runtime state of the servers
	// If we fail to retrieve it, we log the error and consider the serverset not ready
	// This allows the reconciliation to continue in case the ConfigMap is temporarily unavailable
	// If a state map is not configured, we do not care about the value of the stateMap variable at all
	stateMap := &v1.ConfigMap{}
	if cr.Spec.ForProvider.Template.Spec.StateMap != nil {
		if err = e.kube.Get(ctx, types.NamespacedName{
			Name:      cr.Spec.ForProvider.Template.Spec.StateMap.Name,
			Namespace: cr.Spec.ForProvider.Template.Spec.StateMap.Namespace,
		}, stateMap,
		); err != nil {
			e.log.Info(
				"failed to retrieve state ConfigMap for serverset, sset is not ready",
				"name", cr.Name, "stateMap", cr.Spec.ForProvider.Template.Spec.StateMap.Name,
				"namespace", cr.Spec.ForProvider.Template.Spec.StateMap.Namespace, "error", err,
			)
			// Reset stateMap to nil, so that we can differentiate between not found and empty ConfigMap
			stateMap = nil
		}
	}

	areServersCreated := len(servers) == cr.Spec.ForProvider.Replicas
	areServersUpToDate, areServersAvailable, err := AreServersReady(cr.Spec.ForProvider.Template.Spec, servers, stateMap, e.log)
	if err != nil {
		e.log.Info("failed to check if the servers are available and up-to-date", "name", cr.Name, "error", err)
		return managed.ExternalObservation{}, fmt.Errorf("failed to check if servers are available and up-to-date: %w", err)
	}

	volumes, err := GetVolumesOfSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	areBootVolumesCreated := len(volumes) == cr.Spec.ForProvider.Replicas
	areBootVolumesUpToDate, areBootVolumesAvailable := AreBootVolumesReady(cr.Spec.ForProvider.BootVolumeTemplate, volumes)

	nics, err := GetNICsOfSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	crExpectedNoOfNICs := len(cr.Spec.ForProvider.Template.Spec.NICs) * cr.Spec.ForProvider.Replicas
	areNICsCreated := len(nics) == crExpectedNoOfNICs

	// at the moment we do not check that fields of nics are updated, because nic fields are immutable
	e.log.Info("Observing the ServerSet", "name", cr.Name, "areServersUpToDate", areServersUpToDate, "areBootVolumesUpToDate", areBootVolumesUpToDate, "areServersCreated",
		areServersCreated, "areBootVolumesCreated", areBootVolumesCreated, "areNICsCreated", areNICsCreated, "areServersAvailable", areServersAvailable, "areBootVolumesAvailable", areBootVolumesAvailable)
	if areServersAvailable && areBootVolumesAvailable {
		cr.SetConditions(xpv1.Available())
	} else {
		cr.SetConditions(xpv1.Creating())
	}

	return managed.ExternalObservation{
		// Return false when the externalServerSet resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: areServersCreated && areNICsCreated && areBootVolumesCreated,

		// Return false when the externalServerSet resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: areServersUpToDate && areBootVolumesUpToDate,

		// Return any details that may be required to connect to the externalServerSet
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func didNrOfReplicasChange(cr *v1alpha1.ServerSet, replicas []v1alpha1.Server) bool {
	return len(replicas) != cr.Status.AtProvider.Replicas
}

type substitutionConfig struct {
	identities map[string]string
	name       string
	namespace  string
}

func (e *external) populateReplicasStatuses(ctx context.Context, cr *v1alpha1.ServerSet, serverSetReplicas []v1alpha1.Server) {
	if cr.Status.AtProvider.ReplicaStatuses == nil || didNrOfReplicasChange(cr, serverSetReplicas) {
		cr.Status.AtProvider.ReplicaStatuses = make([]v1alpha1.ServerSetReplicaStatus, len(serverSetReplicas))
	}
	for i := range serverSetReplicas {
		replicaStatus := computeStatus(serverSetReplicas[i].Status.AtProvider.State)
		errMsg := ""

		lastCondition := getLastCondition(serverSetReplicas[i])
		if lastCondition.Reason == xpv1.ReasonReconcileError {
			replicaStatus = statusError
			errMsg = lastCondition.Message
		}

		replicaIdx := ComputeReplicaIdx(e.log, fmt.Sprintf(indexLabel, cr.Name, ResourceServer), serverSetReplicas[i].Labels)
		volumeVersion, err := getVolumeVersion(ctx, e.kube, cr.GetName(), replicaIdx)
		if err != nil {
			e.log.Info("error fetching volume version for", "name", cr.GetName(), "replicaIndex", replicaIdx, "error", err)
		}
		nicStatues := computeNicStatuses(ctx, e, cr.Name, replicaIdx)
		cr.Status.AtProvider.ReplicaStatuses[i] = v1alpha1.ServerSetReplicaStatus{
			Role:         fetchRole(ctx, e, *cr, replicaIdx, serverSetReplicas[i].Name, replicaStatus),
			Name:         serverSetReplicas[i].Name,
			ReplicaIndex: replicaIdx,
			NICStatuses:  nicStatues,
			Status:       replicaStatus,
			ErrorMessage: errMsg,
			Hostname:     getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, replicaIdx, volumeVersion),
			LastModified: metav1.Now(),
		}
		// for nfs we need to store substitutions in a configmap(created when the bootvolumes are created) and display them in the status
		if len(cr.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions) > 0 {
			// re-initialize in case of crossplane reboots
			if cr.Status.AtProvider.ReplicaStatuses[i].SubstitutionReplacement == nil {
				cr.Status.AtProvider.ReplicaStatuses[i].SubstitutionReplacement = make(map[string]string, len(cr.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions))
			}
			e.setSubstitutions(ctx, cr, replicaIdx, i)
		} else {
			e.log.Info("no substitutions found in bootvolume template", "serverset name", cr.Name)
		}
	}
	cr.Status.AtProvider.Replicas = len(serverSetReplicas)
}

// setSubstitutions sets substitutions in status. sets again in globalstate if they got lost in case of reboot.
// reads substitutions from configMap that has serverset name and either the identity config map namespace, or default
func (e *external) setSubstitutions(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, sliceIndex int) {
	if len(cr.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions) > 0 {
		volumeVersion, _, _ := getVersionsFromVolumeAndServer(ctx, e.kube, cr.GetName(), replicaIndex)
		for _, subst := range cr.Spec.ForProvider.BootVolumeTemplate.Spec.Substitutions {

			namespace := "default"
			if cr.Spec.ForProvider.IdentityConfigMap.Namespace != "" {
				namespace = cr.Spec.ForProvider.IdentityConfigMap.Namespace
			}
			e.configMapController.SetSubstitutionConfigMap(cr.Name, namespace)

			value := e.configMapController.FetchSubstitutionFromMap(ctx, cr.Name, subst.Key, replicaIndex, volumeVersion)
			if value != "" {
				cr.Status.AtProvider.ReplicaStatuses[sliceIndex].SubstitutionReplacement[subst.Key] = value
				identifier := substitution.Identifier(getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, volumeVersion))
				if _, ok := globalStateMap[cr.Name]; !ok {
					globalStateMap[cr.Name] = substitution.GlobalState{}
				}
				if !globalStateMap[cr.Name].Exists(identifier, subst.Key) {
					globalStateMap[cr.Name].Set(identifier, subst.Key, value)
					e.log.Info("substitution value updated in global state", "serverset name", cr.Name, "for key", subst.Key, "and value", value)
				}
			} else {
				e.log.Info("substitution value not found", "serverset name", cr.Name, "for key", subst.Key)
			}
		}
	}
}

func computeNicStatuses(ctx context.Context, e *external, crName string, replicaIndex int) []v1alpha1.NicStatus {
	nicsOfReplica := &v1alpha1.NicList{}
	err := ListResFromSSetWithIndex(ctx, e.kube, crName, resourceNIC, replicaIndex, nicsOfReplica)
	if err != nil {
		e.log.Info("error fetching nics", "name", crName, "replicaIndex", replicaIndex, "error", err)
		return []v1alpha1.NicStatus{}
	}

	nicStatuses := make([]v1alpha1.NicStatus, len(nicsOfReplica.Items))
	for i, nic := range nicsOfReplica.Items {
		nicStatuses[i] = nic.Status
	}

	return nicStatuses
}

func getLastCondition(server v1alpha1.Server) xpv1.Condition {
	noOfConditions := len(server.Status.Conditions)
	if noOfConditions > 0 {
		return server.Status.Conditions[noOfConditions-1]
	}
	return xpv1.Condition{}
}

func fetchRole(ctx context.Context, e *external, sset v1alpha1.ServerSet, replicaIndex int, replicaName, replicaStatus string) v1alpha1.Role {
	role := v1alpha1.Passive
	if replicaStatus != statusReady {
		return role
	}
	if sset.Spec.ForProvider.IdentityConfigMap.Namespace == "" ||
		sset.Spec.ForProvider.IdentityConfigMap.Name == "" ||
		sset.Spec.ForProvider.IdentityConfigMap.KeyName == "" {
		e.log.Info("no identity configmap values provided, setting role based on replica index only for", "serverset name", sset.Name)
		if replicaIndex == 0 {
			return v1alpha1.Active
		}
		return role
	}
	namespace := sset.Spec.ForProvider.IdentityConfigMap.Namespace
	name := sset.Spec.ForProvider.IdentityConfigMap.Name
	key := sset.Spec.ForProvider.IdentityConfigMap.KeyName
	cfgLease := &v1.ConfigMap{}
	err := e.kube.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cfgLease)
	if err != nil {
		e.log.Info("error fetching config lease, will default to PASSIVE role", "serverset name", sset.Name, "error", err)
		return v1alpha1.Passive
	}

	if cfgLease.Data[key] == replicaName {
		return v1alpha1.Active
	}

	// if it is not in the config map then it has Passive role
	return role
}

func computeStatus(state string) string {
	// At the moment we compute the status of the Server contained in the ServerSet
	// based on the status of the Server.
	switch state {
	case ionoscloud.Available:
		return statusReady
	case ionoscloud.Failed:
		return statusError
	case ionoscloud.Busy:
		return statusBusy
	}
	return statusUnknown
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	cr.Status.SetConditions(xpv1.Creating())

	// for n times of cr.Spec.Replicas, create a server
	// for each server, create a volume
	e.log.Info("Creating a new ServerSet", "name", cr.Name, "replicas", cr.Spec.ForProvider.Replicas)
	for i := 0; i < cr.Spec.ForProvider.Replicas; i++ {
		volumeVersion, serverVersion, err := getVersionsFromVolumeAndServer(ctx, e.kube, cr.GetName(), i)
		if err != nil && !errors.Is(err, errNoVolumesFound) {
			return managed.ExternalCreation{}, err
		}
		if err := e.ensureServerAndNicByIndex(ctx, cr, i, serverVersion); err != nil {
			return managed.ExternalCreation{}, err
		}

		if err := e.ensureBootVolumeByIndex(ctx, cr, i, volumeVersion); err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("while ensuring bootVolume (%w)", err)
		}
		if err := e.attachBootVolume(ctx, cr, i, serverVersion, volumeVersion); err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("while attaching volume to server (%w)", err)
		}
	}

	// When all conditions are met, the managed resource is considered available
	meta.SetExternalName(cr, cr.Name)
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) attachBootVolume(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, serverVersion, volumeVersion int) error {
	bootVolume, err := e.bootVolumeController.Get(ctx, getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, replicaIndex, volumeVersion), cr.Namespace)
	if err != nil {
		return err
	}

	server, err := e.serverController.Get(ctx, getNameFrom(cr.Spec.ForProvider.Template.Metadata.Name, replicaIndex, serverVersion), cr.Namespace)
	if err != nil {
		return err
	}
	server.Spec.ForProvider.VolumeCfg.VolumeID = bootVolume.Status.AtProvider.VolumeID
	return e.serverController.Update(ctx, server)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalUpdate{}, nil
	}
	// how do we know if we want to update servers or nic params?
	err := e.updateServersFromTemplate(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.reconcileVolumesFromTemplate(ctx, cr); err != nil {
		return managed.ExternalUpdate{}, err
	}
	servers, err := GetServersOfSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	e.populateReplicasStatuses(ctx, cr, servers)
	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// externalServerSet resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) updateServersFromTemplate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	servers, err := GetServersOfSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return err
	}
	for idx := range servers {
		// Retrieve the boot-volume associated with the server, so we can check hotplug settings for CPU/RAM changes
		volumeVersion, err := getVolumeVersion(ctx, e.kube, cr.GetName(), idx)
		if err != nil {
			return fmt.Errorf("error getting boot volume version for server %s: %w", servers[idx].Name, err)
		}
		bootVolumeName := getNameFrom(cr.Spec.ForProvider.BootVolumeTemplate.Metadata.Name, idx, volumeVersion)
		bootVolume := &v1alpha1.Volume{}
		if err := e.kube.Get(ctx, types.NamespacedName{
			Name:      bootVolumeName,
			Namespace: cr.Namespace,
		}, bootVolume); err != nil {
			return fmt.Errorf("error getting boot volume %s to check hotplug settings to server %s: %w", bootVolumeName, servers[idx].Name, err)
		}

		update, failover := checkServerDiff(&servers[idx], cr, bootVolume)
		e.log.Info("Checking server for update", "serverset", cr.Name, "server", servers[idx].Name, "update", update, "failover", failover)
		if update {
			requestTimestamp := time.Now()
			if err := e.kube.Update(ctx, &servers[idx]); err != nil {
				return fmt.Errorf("error updating server %w", err)
			}

			if failover {
				e.log.Info("Server requires failover, waiting for update to finish before continuing", "serverset", cr.Name, "server", servers[idx].Name)
				if err := kube.WaitForResource(
					ctx, kube.ResourceReadyTimeout, func(ctx context.Context, name, namespace string) (bool, error) {
						return e.isUpdateFinished(ctx, requestTimestamp, name, namespace)
					}, servers[idx].Name, servers[idx].Namespace,
				); err != nil {
					return fmt.Errorf("error waiting for server to be updated: %w", err)
				}

				if cr.Spec.ForProvider.Template.Spec.StateMap == nil {
					e.log.Info("Successfully updated server", "serverset", cr.Name, "server", servers[idx].Name)
					continue
				}

				e.log.Info("Server has been updated and uses custom state map, waiting for reboot to finish", "serverset", cr.Name, "server", servers[idx].Name)
				if err := kube.WaitForResource(
					ctx, kube.ResourceReadyTimeout, func(ctx context.Context, mapName, mapNamespace string) (bool, error) {
						return e.isVMSoftwareRunning(ctx, requestTimestamp, servers[idx].Name, mapName, mapNamespace)
					}, cr.Spec.ForProvider.Template.Spec.StateMap.Name, cr.Spec.ForProvider.Template.Spec.StateMap.Namespace,
				); err != nil {
					return fmt.Errorf("error waiting for server reboot: %w", err)
				}
			}

			e.log.Info("Successfully updated server", "serverset", cr.Name, "server", servers[idx].Name)
		}
	}
	return nil
}

// reconcileVolumesFromTemplate updates bootvolume, or deletes and re-creates server, volume and nic if something
// immutable changes in a bootvolume
func (e *external) reconcileVolumesFromTemplate(ctx context.Context, cr *v1alpha1.ServerSet) error {
	volumes, err := GetVolumesOfSSet(ctx, e.kube, cr.Name)
	if err != nil {
		return err
	}
	masterIndex := getIdentityFromStatus(cr.Status.AtProvider.ReplicaStatuses)
	err = e.updateOrRecreateVolumes(ctx, cr, volumes, masterIndex)
	if err != nil {
		return fmt.Errorf("while updating volumes for serverset %s %w", cr.Name, err)
	}
	return nil
}

func getIdentityFromStatus(statuses []v1alpha1.ServerSetReplicaStatus) int {
	for idx := range statuses {
		if statuses[idx].Role == v1alpha1.Active {
			return idx
		}
	}
	return -1
}

func (e *external) updateOrRecreateVolumes(ctx context.Context, cr *v1alpha1.ServerSet, volumes []v1alpha1.Volume, masterIndex int) error {
	recreateLeader := false
	for idx := range volumes {
		update := false
		deleteAndCreate := false
		update, deleteAndCreate = updateOrRecreate(&volumes[idx].Spec.ForProvider, cr.Spec.ForProvider.BootVolumeTemplate.Spec)
		if deleteAndCreate {
			// we want to recreate master at the end
			if masterIndex == idx {
				recreateLeader = true
				continue
			}
			err := e.updateByIndex(ctx, idx, cr)
			if err != nil {
				return err
			}
			// we want to return here to be able to update the status before we move to the next bootvolume to update
			return nil
		} else if update {
			if err := e.kube.Update(ctx, &volumes[idx]); err != nil {
				return fmt.Errorf("error updating volume %w", err)
			}
		}
	}
	if masterIndex != -1 {
		e.log.Info("updating leader", "serverset", cr.Name, "index", masterIndex, "template", cr.Spec.ForProvider.BootVolumeTemplate.Spec)
		if recreateLeader {
			err := e.updateByIndex(ctx, masterIndex, cr)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *external) updateByIndex(ctx context.Context, idx int, cr *v1alpha1.ServerSet) error {
	volumeVersion, serverVersion, err := getVersionsFromVolumeAndServer(ctx, e.kube, cr.GetName(), idx)
	if err != nil {
		return err
	}
	// reset globalstate before updating so we get the same ips
	// globalStateMap[cr.Name] = substitution.GlobalState{}
	updater := e.getUpdaterByStrategy(cr.Spec.ForProvider.BootVolumeTemplate.Spec.UpdateStrategy.Stype)
	return updater.update(ctx, cr, idx, volumeVersion, serverVersion)
}

func (e *external) getUpdaterByStrategy(strategyType v1alpha1.UpdateStrategyType) updater {
	switch strategyType {
	case v1alpha1.CreateAllBeforeDestroy:
		return newCreateBeforeDestroy(e.bootVolumeController, e.serverController, e.nicController, e.firewallRuleController)
	case v1alpha1.CreateBeforeDestroyBootVolume:
		return newCreateBeforeDestroyOnlyBootVolume(e.bootVolumeController, e.serverController)
	default:
		return newCreateBeforeDestroyOnlyBootVolume(e.bootVolumeController, e.serverController)
	}
}

// updateOrRecreate checks if bootvolume parameters are equal to bootvolume template parameters
// mutates volume parameters if fields are not equal
func updateOrRecreate(volumeParams *v1alpha1.VolumeParameters, volumeSpec v1alpha1.ServerSetBootVolumeSpec) (update bool, deleteAndCreate bool) {
	if volumeParams.Size != volumeSpec.Size {
		update = true
		volumeParams.Size = volumeSpec.Size
	}
	if volumeParams.Type != volumeSpec.Type {
		deleteAndCreate = true
		volumeParams.Type = volumeSpec.Type
	}

	if volumeParams.Image != volumeSpec.Image {
		deleteAndCreate = true
		volumeParams.Image = volumeSpec.Image
	}
	if volumeParams.SetHotPlugsFromImage != volumeSpec.SetHotPlugsFromImage {
		deleteAndCreate = true
		volumeParams.SetHotPlugsFromImage = volumeSpec.SetHotPlugsFromImage
	}

	return update, deleteAndCreate
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ServerSet)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errUnexpectedObject)
	}
	e.log.Info("Deleting the ServerSet", "name", cr.Name)
	cr.SetConditions(xpv1.Deleting())

	e.log.Info("Deleting the substitution configmap", "name", cr.Name)
	globalStateMap[cr.Name] = substitution.GlobalState{}
	delete(globalStateMap, cr.Name)
	if err := e.configMapController.Delete(ctx, cr.Name); err != nil {
		return managed.ExternalDelete{}, err
	}
	e.log.Info("Finished deleting the ServerSet", "name", cr.Name)

	return managed.ExternalDelete{}, nil
}

// AreServersReady checks if replicas and template params are equal to server obj params
func AreServersReady(
	templateParams v1alpha1.ServerSetTemplateSpec, servers []v1alpha1.Server, stateMap *v1.ConfigMap, log logging.Logger,
) (areServersUpToDate, areServersAvailable bool, err error) {
	for _, serverObj := range servers {
		if serverObj.Spec.ForProvider.Cores != templateParams.Cores {
			return false, false, nil
		}
		if serverObj.Spec.ForProvider.RAM != templateParams.RAM {
			return false, false, nil
		}
		if serverObj.Spec.ForProvider.NicMultiQueue != templateParams.NicMultiQueue {
			return false, false, nil
		}
		if serverObj.Spec.ForProvider.CPUFamily != templateParams.CPUFamily {
			return false, false, nil
		}
		if serverObj.Status.AtProvider.State != ionoscloud.Available {
			return true, false, nil
		}

		if templateParams.StateMap == nil {
			continue
		}

		// Since we allow reconciliation to continue if state map is not found, we check for nil here to ensure
		// that the server is not considered ready in this case
		// We log this info directly in the Observe method
		if stateMap == nil {
			return true, false, nil
		}

		runtimeState, err := checkRuntimeState(*stateMap, serverObj.Name, nil, log)
		if err != nil || !runtimeState {
			return true, runtimeState, err
		}
	}

	return true, true, nil
}

// AreBootVolumesReady checks if template params are equal to volume obj params
func AreBootVolumesReady(templateParams v1alpha1.BootVolumeTemplate, volumes []v1alpha1.Volume) (bool, bool) {
	for _, volumeObj := range volumes {
		if volumeObj.Spec.ForProvider.Size != templateParams.Spec.Size {
			return false, false
		}
		if volumeObj.Spec.ForProvider.Image != templateParams.Spec.Image {
			return false, false
		}
		if volumeObj.Spec.ForProvider.Type != templateParams.Spec.Type {
			return false, false
		}
		if volumeObj.Spec.ForProvider.SetHotPlugsFromImage != templateParams.Spec.SetHotPlugsFromImage {
			return false, false
		}

		if volumeObj.Status.AtProvider.State != ionoscloud.Available {
			return true, false
		}
	}

	return true, true
}

// GetServersOfSSet - gets servers from a server set based on the serverset label
func GetServersOfSSet(ctx context.Context, kube client.Client, name string) ([]v1alpha1.Server, error) {
	serverList := &v1alpha1.ServerList{}
	if err := kube.List(ctx, serverList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return serverList.Items, nil
}

// GetVolumesOfSSet - gets volumes from a server set based on the serverset label
func GetVolumesOfSSet(ctx context.Context, kube client.Client, name string) ([]v1alpha1.Volume, error) {
	volumeList := &v1alpha1.VolumeList{}
	if err := kube.List(ctx, volumeList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return volumeList.Items, nil
}

// GetNICsOfSSet - gets all volumes of a server set
func GetNICsOfSSet(ctx context.Context, kube client.Client, name string) ([]v1alpha1.Nic, error) {
	nicList := &v1alpha1.NicList{}
	if err := kube.List(ctx, nicList, client.MatchingLabels{
		serverSetLabel: name,
	}); err != nil {
		return nil, err
	}

	return nicList.Items, nil
}

// ListResFromSSetWithIndex - lists resources from a server set with a specific index label
func ListResFromSSetWithIndex(ctx context.Context, kube client.Client, serversetName, resType string, index int, list client.ObjectList) error {
	label := client.MatchingLabels{
		fmt.Sprintf(indexLabel, serversetName, resType): strconv.Itoa(index),
	}
	return kube.List(ctx, list, label)
}

// listResFromSSetWithIndexAndVersion - lists resources from a server set with a specific index and version label
func listResFromSSetWithIndexAndVersion(ctx context.Context, kube client.Client, serversetName, resType string, index, version int, list client.ObjectList) error {
	return kube.List(ctx, list, client.MatchingLabels{
		fmt.Sprintf(versionLabel, serversetName, resType): strconv.Itoa(version),
		fmt.Sprintf(indexLabel, serversetName, resType):   strconv.Itoa(index),
	})
}

var errNoVolumesFound = errors.New("no volumes found")

// getVersionsFromVolumeAndServer checks that there is only one server and volume and returns their version
func getVersionsFromVolumeAndServer(ctx context.Context, kube client.Client, serversetName string, replicaIndex int) (volumeVersion int, serverVersion int, err error) {
	volumeVersion, err = getVolumeVersion(ctx, kube, serversetName, replicaIndex)
	if err != nil {
		return volumeVersion, serverVersion, err
	}

	serverVersion, err = getServerVersion(ctx, kube, serversetName, replicaIndex)
	if err != nil {
		return volumeVersion, serverVersion, err
	}
	return volumeVersion, serverVersion, nil
}

func getServerVersion(ctx context.Context, kube client.Client, serversetName string, replicaIndex int) (int, error) {
	serverVersion := 0
	serverResources := &v1alpha1.ServerList{}
	err := ListResFromSSetWithIndex(ctx, kube, serversetName, ResourceServer, replicaIndex, serverResources)
	if err != nil {
		return serverVersion, err
	}
	if len(serverResources.Items) > 1 {
		return serverVersion, fmt.Errorf("found too many servers for index %d", replicaIndex)
	}
	if len(serverResources.Items) == 0 {
		return serverVersion, fmt.Errorf("for index %d %w", replicaIndex, errNoVolumesFound)
	}
	server := serverResources.Items[0]
	return strconv.Atoi(server.Labels[fmt.Sprintf(versionLabel, serversetName, ResourceServer)])
}

func getVolumeVersion(ctx context.Context, kube client.Client, serversetName string, replicaIndex int) (int, error) {
	volumeVersion := 0
	volumeResources := &v1alpha1.VolumeList{}
	err := ListResFromSSetWithIndex(ctx, kube, serversetName, resourceBootVolume, replicaIndex, volumeResources)
	if err != nil {
		return volumeVersion, err
	}
	if len(volumeResources.Items) > 1 {
		return volumeVersion, fmt.Errorf("found too many volumes for index %d", replicaIndex)
	}
	if len(volumeResources.Items) == 0 {
		return volumeVersion, fmt.Errorf("for index %d %w", replicaIndex, errNoVolumesFound)
	}
	volume := volumeResources.Items[0]
	return strconv.Atoi(volume.Labels[fmt.Sprintf(versionLabel, serversetName, resourceBootVolume)])
}

func (e *external) ensureServerAndNicByIndex(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	resSrv := &v1alpha1.ServerList{}
	if err := ListResFromSSetWithIndex(ctx, e.kube, cr.GetName(), ResourceServer, replicaIndex, resSrv); err != nil {
		return err
	}
	if len(resSrv.Items) > 1 {
		return fmt.Errorf("found too many servers for index %d", replicaIndex)
	}
	if len(resSrv.Items) == 0 {
		res := &v1alpha1.VolumeList{}
		volumeVersion := version
		if err := ListResFromSSetWithIndex(ctx, e.kube, cr.GetName(), resourceBootVolume, replicaIndex, res); err != nil {
			return err
		}
		if len(res.Items) > 0 {
			var err error
			volumeVersion, err = strconv.Atoi(res.Items[0].Labels[fmt.Sprintf(versionLabel, cr.GetName(), resourceBootVolume)])
			if err != nil {
				return err
			}
		}

		if err := e.serverController.Ensure(ctx, cr, replicaIndex, version, volumeVersion); err != nil {
			return err
		}

		// refresh the server list after creation
		if err := ListResFromSSetWithIndex(ctx, e.kube, cr.GetName(), ResourceServer, replicaIndex, resSrv); err != nil {
			return err
		}
	}

	if len(resSrv.Items) > 0 {
		serverID := resSrv.Items[0].Status.AtProvider.ServerID
		if serverID == "" {
			_ = e.serverController.Delete(ctx, resSrv.Items[0].Name, cr.Namespace)
			return fmt.Errorf(
				"server creation went wrong, serverID is empty for replica %d of serverset %s, attempting to recreate",
				replicaIndex, cr.Name,
			)
		}

		if err := e.nicController.EnsureNICs(ctx, cr, replicaIndex, version, serverID); err != nil {
			return err
		}

		if err := e.firewallRuleController.EnsureFirewallRules(ctx, cr, replicaIndex, version, serverID); err != nil {
			return err
		}
	}

	return nil
}

// ensureBootVolumeByIndex - ensures boot volume created for a specific index. After checking for index, it checks for index and version
func (e *external) ensureBootVolumeByIndex(ctx context.Context, cr *v1alpha1.ServerSet, replicaIndex, version int) error {
	res := &v1alpha1.VolumeList{}
	if err := ListResFromSSetWithIndex(ctx, e.kube, cr.GetName(), resourceBootVolume, replicaIndex, res); err != nil {
		return err
	}
	if len(res.Items) > 1 {
		return fmt.Errorf("found too many volumes for index %d", replicaIndex)
	}
	if len(res.Items) == 0 {
		if err := e.bootVolumeController.Ensure(ctx, cr, replicaIndex, version); err != nil {
			return err
		}
	}
	return nil
}

// getNameFrom - generates name for a resource
func getNameFrom(resourceName string, idx, version int) string {
	return fmt.Sprintf("%s-%d-%d", resourceName, idx, version)
}

// ComputeReplicaIdx - extracts the replica index from the labels
func ComputeReplicaIdx(log logging.Logger, idxLabel string, labels map[string]string) int {
	idxLabelValue := labels[idxLabel]
	replicaIdx, err := strconv.Atoi(idxLabelValue)
	if err != nil {
		log.Info("could not compute replica index", "error", err, "idxLabelValue", idxLabelValue, "idxLabel", idxLabel)
		return -1
	}
	return replicaIdx
}

// Disconnect does nothing because there are no resources to release. Needs to be implemented starting from crossplane-runtime v0.17
func (e *external) Disconnect(_ context.Context) error {
	return nil
}

// checkServerDiff checks if server parameters are equal to template parameters to decide if an update is needed, as well as if a failover is needed.
// To determine if the failover mechanism needs to be triggered, it checks the hotplug settings of the boot volume for the CPU/RAM.
// If hotplug is disabled for either CPU or RAM and there is a change in the respective parameter, failover is required and is set to true.
// The function mutates the server parameters if they are not equal to the template parameters, so that the server can be updated afterwards.
func checkServerDiff(old *v1alpha1.Server, cr *v1alpha1.ServerSet, bootVolume *v1alpha1.Volume) (update, failover bool) {
	if old.Spec.ForProvider.RAM != cr.Spec.ForProvider.Template.Spec.RAM {
		update = true
		old.Spec.ForProvider.RAM = cr.Spec.ForProvider.Template.Spec.RAM
		if !bootVolume.Spec.ForProvider.RAMHotPlug {
			failover = true
		}
	}
	if old.Spec.ForProvider.Cores != cr.Spec.ForProvider.Template.Spec.Cores {
		update = true
		old.Spec.ForProvider.Cores = cr.Spec.ForProvider.Template.Spec.Cores
		if !bootVolume.Spec.ForProvider.CPUHotPlug {
			failover = true
		}
	}
	if old.Spec.ForProvider.CPUFamily != cr.Spec.ForProvider.Template.Spec.CPUFamily {
		update = true
		old.Spec.ForProvider.CPUFamily = cr.Spec.ForProvider.Template.Spec.CPUFamily
		if !bootVolume.Spec.ForProvider.CPUHotPlug {
			failover = true
		}
	}
	if old.Spec.ForProvider.NicMultiQueue != cr.Spec.ForProvider.Template.Spec.NicMultiQueue {
		update = true
		old.Spec.ForProvider.NicMultiQueue = cr.Spec.ForProvider.Template.Spec.NicMultiQueue
	}

	return update, failover
}

// isUpdateFinished checks the update condition of a server to see if it has been updated after a specific timestamp
// and if it was successful. If the update was processed after the requestTimestamp and was successful, it returns true.
// If the update was processed after the requestTimestamp but failed, it returns an error.
func (e *external) isUpdateFinished(ctx context.Context, requestTimestamp time.Time, name, namespace string) (bool, error) {
	server := &v1alpha1.Server{}
	if err := e.kube.Get(
		ctx, types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}, server,
	); err != nil {
		return false, fmt.Errorf("error getting server %s to check request: %w", name, err)
	}

	updateCondition := server.GetCondition(control.UpdatedConditionType)
	wasProcessed := updateCondition.LastTransitionTime.After(requestTimestamp)
	wasSuccessful := updateCondition.Status == v1.ConditionTrue
	if wasProcessed && !wasSuccessful {
		return false, fmt.Errorf("server %s update failed: %s", name, updateCondition.Message)
	}
	return wasProcessed && wasSuccessful, nil
}

// isVMSoftwareRunning checks the state ConfigMap to see if the server has rebooted successfully and the software on it is back in a running state.
func (e *external) isVMSoftwareRunning(ctx context.Context, requestTimestamp time.Time, serverName, mapName, mapNamespace string) (bool, error) {
	stateMap := v1.ConfigMap{}
	if err := e.kube.Get(
		ctx, types.NamespacedName{
			Name:      mapName,
			Namespace: mapNamespace,
		}, &stateMap,
	); err != nil {
		return false, fmt.Errorf("error getting state map %s (%s) to check running state for server %s: %w", mapName, mapNamespace, serverName, err)
	}

	return checkRuntimeState(stateMap, serverName, &requestTimestamp, e.log)
}

// checkRuntimeState checks the ConfigMap data for the runtime state of a server. If a requestTimestamp is provided, it also checks if the
// state timestamp is after the requestTimestamp.
func checkRuntimeState(stateMap v1.ConfigMap, serverName string, requestTimestamp *time.Time, log logging.Logger) (bool, error) {
	stateKey := fmt.Sprintf(stateKeyFormat, serverName)
	timestampStateKey := fmt.Sprintf(stateTimestampKeyFormat, serverName)

	// If the Data field is nil, it means there is no state information available.
	if stateMap.Data == nil {
		log.Info("state ConfigMap has empty Data", "stateMap", stateMap.Name, "namespace", stateMap.Namespace)
		return false, nil
	}

	// We expect both state and timestamp to be present in the state config map.
	// If either one is missing, it will result in the VM being considered not ready.
	state, ok := stateMap.Data[stateKey]
	if !ok || state == "" {
		log.Info("state key missing in state ConfigMap", "stateKey", stateKey, "stateMap", stateMap.Name, "namespace", stateMap.Namespace)
		return false, nil
	}

	timestampStr, ok := stateMap.Data[timestampStateKey]
	if !ok || timestampStr == "" {
		log.Info("timestamp key missing in state ConfigMap", "timestampStateKey", timestampStateKey, "stateMap", stateMap.Name, "namespace", stateMap.Namespace)
		return false, nil
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return false, fmt.Errorf("error parsing state timeststate for server %s: %w", serverName, err)
	}

	switch {
	case requestTimestamp != nil && requestTimestamp.Before(timestamp):
		return false, nil
	case state == statusVMError:
		return false, fmt.Errorf("server %s is in VM-ERROR runtime state", serverName)
	case state == statusVMRunning:
		return true, nil
	default:
		return false, nil
	}
}
