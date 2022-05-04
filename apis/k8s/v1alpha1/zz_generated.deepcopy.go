//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/crossplane/crossplane-runtime/apis/common/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cluster) DeepCopyInto(out *Cluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cluster.
func (in *Cluster) DeepCopy() *Cluster {
	if in == nil {
		return nil
	}
	out := new(Cluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Cluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterConfig) DeepCopyInto(out *ClusterConfig) {
	*out = *in
	if in.ClusterIDRef != nil {
		in, out := &in.ClusterIDRef, &out.ClusterIDRef
		*out = new(v1.Reference)
		**out = **in
	}
	if in.ClusterIDSelector != nil {
		in, out := &in.ClusterIDSelector, &out.ClusterIDSelector
		*out = new(v1.Selector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterConfig.
func (in *ClusterConfig) DeepCopy() *ClusterConfig {
	if in == nil {
		return nil
	}
	out := new(ClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterList) DeepCopyInto(out *ClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Cluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterList.
func (in *ClusterList) DeepCopy() *ClusterList {
	if in == nil {
		return nil
	}
	out := new(ClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterObservation) DeepCopyInto(out *ClusterObservation) {
	*out = *in
	if in.AvailableUpgradeVersions != nil {
		in, out := &in.AvailableUpgradeVersions, &out.AvailableUpgradeVersions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ViableNodePoolVersions != nil {
		in, out := &in.ViableNodePoolVersions, &out.ViableNodePoolVersions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterObservation.
func (in *ClusterObservation) DeepCopy() *ClusterObservation {
	if in == nil {
		return nil
	}
	out := new(ClusterObservation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterParameters) DeepCopyInto(out *ClusterParameters) {
	*out = *in
	out.MaintenanceWindow = in.MaintenanceWindow
	if in.APISubnetAllowList != nil {
		in, out := &in.APISubnetAllowList, &out.APISubnetAllowList
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.S3Buckets != nil {
		in, out := &in.S3Buckets, &out.S3Buckets
		*out = make([]S3Bucket, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterParameters.
func (in *ClusterParameters) DeepCopy() *ClusterParameters {
	if in == nil {
		return nil
	}
	out := new(ClusterParameters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSpec) DeepCopyInto(out *ClusterSpec) {
	*out = *in
	in.ResourceSpec.DeepCopyInto(&out.ResourceSpec)
	in.ForProvider.DeepCopyInto(&out.ForProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSpec.
func (in *ClusterSpec) DeepCopy() *ClusterSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterStatus) DeepCopyInto(out *ClusterStatus) {
	*out = *in
	in.ResourceStatus.DeepCopyInto(&out.ResourceStatus)
	in.AtProvider.DeepCopyInto(&out.AtProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterStatus.
func (in *ClusterStatus) DeepCopy() *ClusterStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DatacenterConfig) DeepCopyInto(out *DatacenterConfig) {
	*out = *in
	if in.DatacenterIDRef != nil {
		in, out := &in.DatacenterIDRef, &out.DatacenterIDRef
		*out = new(v1.Reference)
		**out = **in
	}
	if in.DatacenterIDSelector != nil {
		in, out := &in.DatacenterIDSelector, &out.DatacenterIDSelector
		*out = new(v1.Selector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DatacenterConfig.
func (in *DatacenterConfig) DeepCopy() *DatacenterConfig {
	if in == nil {
		return nil
	}
	out := new(DatacenterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IPBlockConfig) DeepCopyInto(out *IPBlockConfig) {
	*out = *in
	if in.IPBlockIDRef != nil {
		in, out := &in.IPBlockIDRef, &out.IPBlockIDRef
		*out = new(v1.Reference)
		**out = **in
	}
	if in.IPBlockIDSelector != nil {
		in, out := &in.IPBlockIDSelector, &out.IPBlockIDSelector
		*out = new(v1.Selector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IPBlockConfig.
func (in *IPBlockConfig) DeepCopy() *IPBlockConfig {
	if in == nil {
		return nil
	}
	out := new(IPBlockConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IPConfig) DeepCopyInto(out *IPConfig) {
	*out = *in
	in.IPBlockCfg.DeepCopyInto(&out.IPBlockCfg)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IPConfig.
func (in *IPConfig) DeepCopy() *IPConfig {
	if in == nil {
		return nil
	}
	out := new(IPConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IPsBlockConfig) DeepCopyInto(out *IPsBlockConfig) {
	*out = *in
	if in.IPBlockIDRef != nil {
		in, out := &in.IPBlockIDRef, &out.IPBlockIDRef
		*out = new(v1.Reference)
		**out = **in
	}
	if in.IPBlockIDSelector != nil {
		in, out := &in.IPBlockIDSelector, &out.IPBlockIDSelector
		*out = new(v1.Selector)
		(*in).DeepCopyInto(*out)
	}
	if in.Indexes != nil {
		in, out := &in.Indexes, &out.Indexes
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IPsBlockConfig.
func (in *IPsBlockConfig) DeepCopy() *IPsBlockConfig {
	if in == nil {
		return nil
	}
	out := new(IPsBlockConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IPsConfigs) DeepCopyInto(out *IPsConfigs) {
	*out = *in
	if in.IPs != nil {
		in, out := &in.IPs, &out.IPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.IPBlockCfgs != nil {
		in, out := &in.IPBlockCfgs, &out.IPBlockCfgs
		*out = make([]IPsBlockConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IPsConfigs.
func (in *IPsConfigs) DeepCopy() *IPsConfigs {
	if in == nil {
		return nil
	}
	out := new(IPsConfigs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesAutoScaling) DeepCopyInto(out *KubernetesAutoScaling) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesAutoScaling.
func (in *KubernetesAutoScaling) DeepCopy() *KubernetesAutoScaling {
	if in == nil {
		return nil
	}
	out := new(KubernetesAutoScaling)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesNodePoolLan) DeepCopyInto(out *KubernetesNodePoolLan) {
	*out = *in
	in.LanCfg.DeepCopyInto(&out.LanCfg)
	if in.Routes != nil {
		in, out := &in.Routes, &out.Routes
		*out = make([]KubernetesNodePoolLanRoutes, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesNodePoolLan.
func (in *KubernetesNodePoolLan) DeepCopy() *KubernetesNodePoolLan {
	if in == nil {
		return nil
	}
	out := new(KubernetesNodePoolLan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesNodePoolLanRoutes) DeepCopyInto(out *KubernetesNodePoolLanRoutes) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesNodePoolLanRoutes.
func (in *KubernetesNodePoolLanRoutes) DeepCopy() *KubernetesNodePoolLanRoutes {
	if in == nil {
		return nil
	}
	out := new(KubernetesNodePoolLanRoutes)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LanConfig) DeepCopyInto(out *LanConfig) {
	*out = *in
	if in.LanIDRef != nil {
		in, out := &in.LanIDRef, &out.LanIDRef
		*out = new(v1.Reference)
		**out = **in
	}
	if in.LanIDSelector != nil {
		in, out := &in.LanIDSelector, &out.LanIDSelector
		*out = new(v1.Selector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LanConfig.
func (in *LanConfig) DeepCopy() *LanConfig {
	if in == nil {
		return nil
	}
	out := new(LanConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MaintenanceWindow) DeepCopyInto(out *MaintenanceWindow) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MaintenanceWindow.
func (in *MaintenanceWindow) DeepCopy() *MaintenanceWindow {
	if in == nil {
		return nil
	}
	out := new(MaintenanceWindow)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePool) DeepCopyInto(out *NodePool) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePool.
func (in *NodePool) DeepCopy() *NodePool {
	if in == nil {
		return nil
	}
	out := new(NodePool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NodePool) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePoolList) DeepCopyInto(out *NodePoolList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NodePool, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePoolList.
func (in *NodePoolList) DeepCopy() *NodePoolList {
	if in == nil {
		return nil
	}
	out := new(NodePoolList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NodePoolList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePoolObservation) DeepCopyInto(out *NodePoolObservation) {
	*out = *in
	if in.PublicIPs != nil {
		in, out := &in.PublicIPs, &out.PublicIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AvailableUpgradeVersions != nil {
		in, out := &in.AvailableUpgradeVersions, &out.AvailableUpgradeVersions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePoolObservation.
func (in *NodePoolObservation) DeepCopy() *NodePoolObservation {
	if in == nil {
		return nil
	}
	out := new(NodePoolObservation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePoolParameters) DeepCopyInto(out *NodePoolParameters) {
	*out = *in
	in.ClusterCfg.DeepCopyInto(&out.ClusterCfg)
	in.DatacenterCfg.DeepCopyInto(&out.DatacenterCfg)
	out.MaintenanceWindow = in.MaintenanceWindow
	out.AutoScaling = in.AutoScaling
	if in.Lans != nil {
		in, out := &in.Lans, &out.Lans
		*out = make([]KubernetesNodePoolLan, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.PublicIPsCfg.DeepCopyInto(&out.PublicIPsCfg)
	in.GatewayIPCfg.DeepCopyInto(&out.GatewayIPCfg)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePoolParameters.
func (in *NodePoolParameters) DeepCopy() *NodePoolParameters {
	if in == nil {
		return nil
	}
	out := new(NodePoolParameters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePoolSpec) DeepCopyInto(out *NodePoolSpec) {
	*out = *in
	in.ResourceSpec.DeepCopyInto(&out.ResourceSpec)
	in.ForProvider.DeepCopyInto(&out.ForProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePoolSpec.
func (in *NodePoolSpec) DeepCopy() *NodePoolSpec {
	if in == nil {
		return nil
	}
	out := new(NodePoolSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodePoolStatus) DeepCopyInto(out *NodePoolStatus) {
	*out = *in
	in.ResourceStatus.DeepCopyInto(&out.ResourceStatus)
	in.AtProvider.DeepCopyInto(&out.AtProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodePoolStatus.
func (in *NodePoolStatus) DeepCopy() *NodePoolStatus {
	if in == nil {
		return nil
	}
	out := new(NodePoolStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *S3Bucket) DeepCopyInto(out *S3Bucket) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new S3Bucket.
func (in *S3Bucket) DeepCopy() *S3Bucket {
	if in == nil {
		return nil
	}
	out := new(S3Bucket)
	in.DeepCopyInto(out)
	return out
}
