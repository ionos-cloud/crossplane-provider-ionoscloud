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
func (in *ApplicationLoadBalancer) DeepCopyInto(out *ApplicationLoadBalancer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancer.
func (in *ApplicationLoadBalancer) DeepCopy() *ApplicationLoadBalancer {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationLoadBalancer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationLoadBalancerList) DeepCopyInto(out *ApplicationLoadBalancerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ApplicationLoadBalancer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerList.
func (in *ApplicationLoadBalancerList) DeepCopy() *ApplicationLoadBalancerList {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApplicationLoadBalancerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationLoadBalancerObservation) DeepCopyInto(out *ApplicationLoadBalancerObservation) {
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerObservation.
func (in *ApplicationLoadBalancerObservation) DeepCopy() *ApplicationLoadBalancerObservation {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerObservation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationLoadBalancerParameters) DeepCopyInto(out *ApplicationLoadBalancerParameters) {
	*out = *in
	in.DatacenterCfg.DeepCopyInto(&out.DatacenterCfg)
	in.IpsCfg.DeepCopyInto(&out.IpsCfg)
	if in.LbPrivateIps != nil {
		in, out := &in.LbPrivateIps, &out.LbPrivateIps
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerParameters.
func (in *ApplicationLoadBalancerParameters) DeepCopy() *ApplicationLoadBalancerParameters {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerParameters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationLoadBalancerSpec) DeepCopyInto(out *ApplicationLoadBalancerSpec) {
	*out = *in
	in.ResourceSpec.DeepCopyInto(&out.ResourceSpec)
	in.ForProvider.DeepCopyInto(&out.ForProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerSpec.
func (in *ApplicationLoadBalancerSpec) DeepCopy() *ApplicationLoadBalancerSpec {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationLoadBalancerStatus) DeepCopyInto(out *ApplicationLoadBalancerStatus) {
	*out = *in
	in.ResourceStatus.DeepCopyInto(&out.ResourceStatus)
	in.AtProvider.DeepCopyInto(&out.AtProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerStatus.
func (in *ApplicationLoadBalancerStatus) DeepCopy() *ApplicationLoadBalancerStatus {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerStatus)
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
	if in.Indexes != nil {
		in, out := &in.Indexes, &out.Indexes
		*out = make([]int, len(*in))
		copy(*out, *in)
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
func (in *IpsConfig) DeepCopyInto(out *IpsConfig) {
	*out = *in
	if in.Ips != nil {
		in, out := &in.Ips, &out.Ips
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.IPBlockCfgs != nil {
		in, out := &in.IPBlockCfgs, &out.IPBlockCfgs
		*out = make([]IPBlockConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IpsConfig.
func (in *IpsConfig) DeepCopy() *IpsConfig {
	if in == nil {
		return nil
	}
	out := new(IpsConfig)
	in.DeepCopyInto(out)
	return out
}
