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
func (in *ApplicationLoadBalancerConfig) DeepCopyInto(out *ApplicationLoadBalancerConfig) {
	*out = *in
	if in.ApplicationLoadBalancerIDRef != nil {
		in, out := &in.ApplicationLoadBalancerIDRef, &out.ApplicationLoadBalancerIDRef
		*out = new(v1.Reference)
		**out = **in
	}
	if in.ApplicationLoadBalancerIDSelector != nil {
		in, out := &in.ApplicationLoadBalancerIDSelector, &out.ApplicationLoadBalancerIDSelector
		*out = new(v1.Selector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerConfig.
func (in *ApplicationLoadBalancerConfig) DeepCopy() *ApplicationLoadBalancerConfig {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationLoadBalancerHTTPRule) DeepCopyInto(out *ApplicationLoadBalancerHTTPRule) {
	*out = *in
	in.TargetGroupCfg.DeepCopyInto(&out.TargetGroupCfg)
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ApplicationLoadBalancerHTTPRuleCondition, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerHTTPRule.
func (in *ApplicationLoadBalancerHTTPRule) DeepCopy() *ApplicationLoadBalancerHTTPRule {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerHTTPRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApplicationLoadBalancerHTTPRuleCondition) DeepCopyInto(out *ApplicationLoadBalancerHTTPRuleCondition) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationLoadBalancerHTTPRuleCondition.
func (in *ApplicationLoadBalancerHTTPRuleCondition) DeepCopy() *ApplicationLoadBalancerHTTPRuleCondition {
	if in == nil {
		return nil
	}
	out := new(ApplicationLoadBalancerHTTPRuleCondition)
	in.DeepCopyInto(out)
	return out
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
	in.ListenerLanCfg.DeepCopyInto(&out.ListenerLanCfg)
	in.TargetLanCfg.DeepCopyInto(&out.TargetLanCfg)
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
func (in *ForwardingRule) DeepCopyInto(out *ForwardingRule) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardingRule.
func (in *ForwardingRule) DeepCopy() *ForwardingRule {
	if in == nil {
		return nil
	}
	out := new(ForwardingRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ForwardingRule) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardingRuleList) DeepCopyInto(out *ForwardingRuleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ForwardingRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardingRuleList.
func (in *ForwardingRuleList) DeepCopy() *ForwardingRuleList {
	if in == nil {
		return nil
	}
	out := new(ForwardingRuleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ForwardingRuleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardingRuleObservation) DeepCopyInto(out *ForwardingRuleObservation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardingRuleObservation.
func (in *ForwardingRuleObservation) DeepCopy() *ForwardingRuleObservation {
	if in == nil {
		return nil
	}
	out := new(ForwardingRuleObservation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardingRuleParameters) DeepCopyInto(out *ForwardingRuleParameters) {
	*out = *in
	in.DatacenterCfg.DeepCopyInto(&out.DatacenterCfg)
	in.ALBCfg.DeepCopyInto(&out.ALBCfg)
	in.ListenerIP.DeepCopyInto(&out.ListenerIP)
	if in.ServerCertificatesIDs != nil {
		in, out := &in.ServerCertificatesIDs, &out.ServerCertificatesIDs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.HTTPRules != nil {
		in, out := &in.HTTPRules, &out.HTTPRules
		*out = make([]ApplicationLoadBalancerHTTPRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardingRuleParameters.
func (in *ForwardingRuleParameters) DeepCopy() *ForwardingRuleParameters {
	if in == nil {
		return nil
	}
	out := new(ForwardingRuleParameters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardingRuleSpec) DeepCopyInto(out *ForwardingRuleSpec) {
	*out = *in
	in.ResourceSpec.DeepCopyInto(&out.ResourceSpec)
	in.ForProvider.DeepCopyInto(&out.ForProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardingRuleSpec.
func (in *ForwardingRuleSpec) DeepCopy() *ForwardingRuleSpec {
	if in == nil {
		return nil
	}
	out := new(ForwardingRuleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ForwardingRuleStatus) DeepCopyInto(out *ForwardingRuleStatus) {
	*out = *in
	in.ResourceStatus.DeepCopyInto(&out.ResourceStatus)
	out.AtProvider = in.AtProvider
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ForwardingRuleStatus.
func (in *ForwardingRuleStatus) DeepCopy() *ForwardingRuleStatus {
	if in == nil {
		return nil
	}
	out := new(ForwardingRuleStatus)
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
func (in *TargetGroup) DeepCopyInto(out *TargetGroup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroup.
func (in *TargetGroup) DeepCopy() *TargetGroup {
	if in == nil {
		return nil
	}
	out := new(TargetGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TargetGroup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupConfig) DeepCopyInto(out *TargetGroupConfig) {
	*out = *in
	if in.TargetGroupIDRef != nil {
		in, out := &in.TargetGroupIDRef, &out.TargetGroupIDRef
		*out = new(v1.Reference)
		**out = **in
	}
	if in.TargetGroupIDSelector != nil {
		in, out := &in.TargetGroupIDSelector, &out.TargetGroupIDSelector
		*out = new(v1.Selector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupConfig.
func (in *TargetGroupConfig) DeepCopy() *TargetGroupConfig {
	if in == nil {
		return nil
	}
	out := new(TargetGroupConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupHTTPHealthCheck) DeepCopyInto(out *TargetGroupHTTPHealthCheck) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupHTTPHealthCheck.
func (in *TargetGroupHTTPHealthCheck) DeepCopy() *TargetGroupHTTPHealthCheck {
	if in == nil {
		return nil
	}
	out := new(TargetGroupHTTPHealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupHealthCheck) DeepCopyInto(out *TargetGroupHealthCheck) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupHealthCheck.
func (in *TargetGroupHealthCheck) DeepCopy() *TargetGroupHealthCheck {
	if in == nil {
		return nil
	}
	out := new(TargetGroupHealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupList) DeepCopyInto(out *TargetGroupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TargetGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupList.
func (in *TargetGroupList) DeepCopy() *TargetGroupList {
	if in == nil {
		return nil
	}
	out := new(TargetGroupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TargetGroupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupObservation) DeepCopyInto(out *TargetGroupObservation) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupObservation.
func (in *TargetGroupObservation) DeepCopy() *TargetGroupObservation {
	if in == nil {
		return nil
	}
	out := new(TargetGroupObservation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupParameters) DeepCopyInto(out *TargetGroupParameters) {
	*out = *in
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]TargetGroupTarget, len(*in))
		copy(*out, *in)
	}
	out.HealthCheck = in.HealthCheck
	out.HTTPHealthCheck = in.HTTPHealthCheck
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupParameters.
func (in *TargetGroupParameters) DeepCopy() *TargetGroupParameters {
	if in == nil {
		return nil
	}
	out := new(TargetGroupParameters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupSpec) DeepCopyInto(out *TargetGroupSpec) {
	*out = *in
	in.ResourceSpec.DeepCopyInto(&out.ResourceSpec)
	in.ForProvider.DeepCopyInto(&out.ForProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupSpec.
func (in *TargetGroupSpec) DeepCopy() *TargetGroupSpec {
	if in == nil {
		return nil
	}
	out := new(TargetGroupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupStatus) DeepCopyInto(out *TargetGroupStatus) {
	*out = *in
	in.ResourceStatus.DeepCopyInto(&out.ResourceStatus)
	out.AtProvider = in.AtProvider
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupStatus.
func (in *TargetGroupStatus) DeepCopy() *TargetGroupStatus {
	if in == nil {
		return nil
	}
	out := new(TargetGroupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetGroupTarget) DeepCopyInto(out *TargetGroupTarget) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetGroupTarget.
func (in *TargetGroupTarget) DeepCopy() *TargetGroupTarget {
	if in == nil {
		return nil
	}
	out := new(TargetGroupTarget)
	in.DeepCopyInto(out)
	return out
}
