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
// Code generated by angryjet. DO NOT EDIT.

package v1alpha1

import (
	"context"
	reference "github.com/crossplane/crossplane-runtime/pkg/reference"
	v1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/backup/v1alpha1"
	errors "github.com/pkg/errors"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveReferences of this CubeServer.
func (mg *CubeServer) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &DatacenterList{},
			Managed: &Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitID,
		Extract:      v1alpha1.ExtractBackupUnitID(),
		Reference:    mg.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitIDRef,
		Selector:     mg.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitIDSelector,
		To: reference.To{
			List:    &v1alpha1.BackupUnitList{},
			Managed: &v1alpha1.BackupUnit{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitID")
	}
	mg.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitID = rsp.ResolvedValue
	mg.Spec.ForProvider.DasVolumeProperties.BackupUnitCfg.BackupUnitIDRef = rsp.ResolvedReference

	return nil
}

// ResolveReferences of this FirewallRule.
func (mg *FirewallRule) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &DatacenterList{},
			Managed: &Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.ServerCfg.ServerID,
		Extract:      ExtractServerID(),
		Reference:    mg.Spec.ForProvider.ServerCfg.ServerIDRef,
		Selector:     mg.Spec.ForProvider.ServerCfg.ServerIDSelector,
		To: reference.To{
			List:    &ServerList{},
			Managed: &Server{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.ServerCfg.ServerID")
	}
	mg.Spec.ForProvider.ServerCfg.ServerID = rsp.ResolvedValue
	mg.Spec.ForProvider.ServerCfg.ServerIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.NicCfg.NicID,
		Extract:      ExtractNicID(),
		Reference:    mg.Spec.ForProvider.NicCfg.NicIDRef,
		Selector:     mg.Spec.ForProvider.NicCfg.NicIDSelector,
		To: reference.To{
			List:    &NicList{},
			Managed: &Nic{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.NicCfg.NicID")
	}
	mg.Spec.ForProvider.NicCfg.NicID = rsp.ResolvedValue
	mg.Spec.ForProvider.NicCfg.NicIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockID,
		Extract:      ExtractIPBlockID(),
		Reference:    mg.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockIDRef,
		Selector:     mg.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockIDSelector,
		To: reference.To{
			List:    &IPBlockList{},
			Managed: &IPBlock{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockID")
	}
	mg.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockID = rsp.ResolvedValue
	mg.Spec.ForProvider.SourceIPCfg.IPBlockCfg.IPBlockIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockID,
		Extract:      ExtractIPBlockID(),
		Reference:    mg.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockIDRef,
		Selector:     mg.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockIDSelector,
		To: reference.To{
			List:    &IPBlockList{},
			Managed: &IPBlock{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockID")
	}
	mg.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockID = rsp.ResolvedValue
	mg.Spec.ForProvider.TargetIPCfg.IPBlockCfg.IPBlockIDRef = rsp.ResolvedReference

	return nil
}

// ResolveReferences of this IPFailover.
func (mg *IPFailover) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &DatacenterList{},
			Managed: &Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.LanCfg.LanID,
		Extract:      ExtractLanID(),
		Reference:    mg.Spec.ForProvider.LanCfg.LanIDRef,
		Selector:     mg.Spec.ForProvider.LanCfg.LanIDSelector,
		To: reference.To{
			List:    &LanList{},
			Managed: &Lan{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.LanCfg.LanID")
	}
	mg.Spec.ForProvider.LanCfg.LanID = rsp.ResolvedValue
	mg.Spec.ForProvider.LanCfg.LanIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.NicCfg.NicID,
		Extract:      ExtractNicID(),
		Reference:    mg.Spec.ForProvider.NicCfg.NicIDRef,
		Selector:     mg.Spec.ForProvider.NicCfg.NicIDSelector,
		To: reference.To{
			List:    &NicList{},
			Managed: &Nic{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.NicCfg.NicID")
	}
	mg.Spec.ForProvider.NicCfg.NicID = rsp.ResolvedValue
	mg.Spec.ForProvider.NicCfg.NicIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockID,
		Extract:      ExtractIPBlockID(),
		Reference:    mg.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockIDRef,
		Selector:     mg.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockIDSelector,
		To: reference.To{
			List:    &IPBlockList{},
			Managed: &IPBlock{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockID")
	}
	mg.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockID = rsp.ResolvedValue
	mg.Spec.ForProvider.IPCfg.IPBlockCfg.IPBlockIDRef = rsp.ResolvedReference

	return nil
}

// ResolveReferences of this Lan.
func (mg *Lan) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &DatacenterList{},
			Managed: &Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.Pcc.PrivateCrossConnectID,
		Extract:      ExtractPccID(),
		Reference:    mg.Spec.ForProvider.Pcc.PrivateCrossConnectIDRef,
		Selector:     mg.Spec.ForProvider.Pcc.PrivateCrossConnectIDSelector,
		To: reference.To{
			List:    &PccList{},
			Managed: &Pcc{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.Pcc.PrivateCrossConnectID")
	}
	mg.Spec.ForProvider.Pcc.PrivateCrossConnectID = rsp.ResolvedValue
	mg.Spec.ForProvider.Pcc.PrivateCrossConnectIDRef = rsp.ResolvedReference

	return nil
}

// ResolveReferences of this ManagementGroup.
func (mg *ManagementGroup) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	for i3 := 0; i3 < len(mg.Spec.ForProvider.UserCfg); i3++ {
		rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
			CurrentValue: mg.Spec.ForProvider.UserCfg[i3].UserID,
			Extract:      ExtractUserID(),
			Reference:    mg.Spec.ForProvider.UserCfg[i3].UserIDRef,
			Selector:     mg.Spec.ForProvider.UserCfg[i3].UserIDSelector,
			To: reference.To{
				List:    &UserList{},
				Managed: &User{},
			},
		})
		if err != nil {
			return errors.Wrap(err, "mg.Spec.ForProvider.UserCfg[i3].UserID")
		}
		mg.Spec.ForProvider.UserCfg[i3].UserID = rsp.ResolvedValue
		mg.Spec.ForProvider.UserCfg[i3].UserIDRef = rsp.ResolvedReference

	}

	return nil
}

// ResolveReferences of this Nic.
func (mg *Nic) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &DatacenterList{},
			Managed: &Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.ServerCfg.ServerID,
		Extract:      ExtractServerID(),
		Reference:    mg.Spec.ForProvider.ServerCfg.ServerIDRef,
		Selector:     mg.Spec.ForProvider.ServerCfg.ServerIDSelector,
		To: reference.To{
			List:    &ServerList{},
			Managed: &Server{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.ServerCfg.ServerID")
	}
	mg.Spec.ForProvider.ServerCfg.ServerID = rsp.ResolvedValue
	mg.Spec.ForProvider.ServerCfg.ServerIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.LanCfg.LanID,
		Extract:      ExtractLanID(),
		Reference:    mg.Spec.ForProvider.LanCfg.LanIDRef,
		Selector:     mg.Spec.ForProvider.LanCfg.LanIDSelector,
		To: reference.To{
			List:    &LanList{},
			Managed: &Lan{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.LanCfg.LanID")
	}
	mg.Spec.ForProvider.LanCfg.LanID = rsp.ResolvedValue
	mg.Spec.ForProvider.LanCfg.LanIDRef = rsp.ResolvedReference

	for i4 := 0; i4 < len(mg.Spec.ForProvider.IpsCfg.IPBlockCfgs); i4++ {
		rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
			CurrentValue: mg.Spec.ForProvider.IpsCfg.IPBlockCfgs[i4].IPBlockID,
			Extract:      ExtractIPBlockID(),
			Reference:    mg.Spec.ForProvider.IpsCfg.IPBlockCfgs[i4].IPBlockIDRef,
			Selector:     mg.Spec.ForProvider.IpsCfg.IPBlockCfgs[i4].IPBlockIDSelector,
			To: reference.To{
				List:    &IPBlockList{},
				Managed: &IPBlock{},
			},
		})
		if err != nil {
			return errors.Wrap(err, "mg.Spec.ForProvider.IpsCfg.IPBlockCfgs[i4].IPBlockID")
		}
		mg.Spec.ForProvider.IpsCfg.IPBlockCfgs[i4].IPBlockID = rsp.ResolvedValue
		mg.Spec.ForProvider.IpsCfg.IPBlockCfgs[i4].IPBlockIDRef = rsp.ResolvedReference

	}

	return nil
}

// ResolveReferences of this Server.
func (mg *Server) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &DatacenterList{},
			Managed: &Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.VolumeCfg.VolumeID,
		Extract:      ExtractVolumeID(),
		Reference:    mg.Spec.ForProvider.VolumeCfg.VolumeIDRef,
		Selector:     mg.Spec.ForProvider.VolumeCfg.VolumeIDSelector,
		To: reference.To{
			List:    &VolumeList{},
			Managed: &Volume{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.VolumeCfg.VolumeID")
	}
	mg.Spec.ForProvider.VolumeCfg.VolumeID = rsp.ResolvedValue
	mg.Spec.ForProvider.VolumeCfg.VolumeIDRef = rsp.ResolvedReference

	return nil
}

// ResolveReferences of this Volume.
func (mg *Volume) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &DatacenterList{},
			Managed: &Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.BackupUnitCfg.BackupUnitID,
		Extract:      v1alpha1.ExtractBackupUnitID(),
		Reference:    mg.Spec.ForProvider.BackupUnitCfg.BackupUnitIDRef,
		Selector:     mg.Spec.ForProvider.BackupUnitCfg.BackupUnitIDSelector,
		To: reference.To{
			List:    &v1alpha1.BackupUnitList{},
			Managed: &v1alpha1.BackupUnit{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.BackupUnitCfg.BackupUnitID")
	}
	mg.Spec.ForProvider.BackupUnitCfg.BackupUnitID = rsp.ResolvedValue
	mg.Spec.ForProvider.BackupUnitCfg.BackupUnitIDRef = rsp.ResolvedReference

	return nil
}
