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
	v1alpha1 "github.com/ionos-cloud/crossplane-provider-ionoscloud/apis/compute/v1alpha1"
	errors "github.com/pkg/errors"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResolveReferences of this NodePool.
func (mg *NodePool) ResolveReferences(ctx context.Context, c client.Reader) error {
	r := reference.NewAPIResolver(c, mg)

	var rsp reference.ResolutionResponse
	var err error

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.ClusterCfg.ClusterID,
		Extract:      ExtractClusterID(),
		Reference:    mg.Spec.ForProvider.ClusterCfg.ClusterIDRef,
		Selector:     mg.Spec.ForProvider.ClusterCfg.ClusterIDSelector,
		To: reference.To{
			List:    &ClusterList{},
			Managed: &Cluster{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.ClusterCfg.ClusterID")
	}
	mg.Spec.ForProvider.ClusterCfg.ClusterID = rsp.ResolvedValue
	mg.Spec.ForProvider.ClusterCfg.ClusterIDRef = rsp.ResolvedReference

	rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: mg.Spec.ForProvider.DatacenterCfg.DatacenterID,
		Extract:      v1alpha1.ExtractDatacenterID(),
		Reference:    mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef,
		Selector:     mg.Spec.ForProvider.DatacenterCfg.DatacenterIDSelector,
		To: reference.To{
			List:    &v1alpha1.DatacenterList{},
			Managed: &v1alpha1.Datacenter{},
		},
	})
	if err != nil {
		return errors.Wrap(err, "mg.Spec.ForProvider.DatacenterCfg.DatacenterID")
	}
	mg.Spec.ForProvider.DatacenterCfg.DatacenterID = rsp.ResolvedValue
	mg.Spec.ForProvider.DatacenterCfg.DatacenterIDRef = rsp.ResolvedReference

	for i3 := 0; i3 < len(mg.Spec.ForProvider.Lans); i3++ {
		rsp, err = r.Resolve(ctx, reference.ResolutionRequest{
			CurrentValue: mg.Spec.ForProvider.Lans[i3].LanCfg.LanID,
			Extract:      v1alpha1.ExtractLanID(),
			Reference:    mg.Spec.ForProvider.Lans[i3].LanCfg.LanIDRef,
			Selector:     mg.Spec.ForProvider.Lans[i3].LanCfg.LanIDSelector,
			To: reference.To{
				List:    &v1alpha1.LanList{},
				Managed: &v1alpha1.Lan{},
			},
		})
		if err != nil {
			return errors.Wrap(err, "mg.Spec.ForProvider.Lans[i3].LanCfg.LanID")
		}
		mg.Spec.ForProvider.Lans[i3].LanCfg.LanID = rsp.ResolvedValue
		mg.Spec.ForProvider.Lans[i3].LanCfg.LanIDRef = rsp.ResolvedReference

	}

	return nil
}