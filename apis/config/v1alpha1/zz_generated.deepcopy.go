//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubelet/config/v1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DynamicCredentialProviderConfig) DeepCopyInto(out *DynamicCredentialProviderConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Mirror != nil {
		in, out := &in.Mirror, &out.Mirror
		*out = new(MirrorConfig)
		**out = **in
	}
	if in.CredentialProviders != nil {
		in, out := &in.CredentialProviders, &out.CredentialProviders
		*out = new(v1.CredentialProviderConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DynamicCredentialProviderConfig.
func (in *DynamicCredentialProviderConfig) DeepCopy() *DynamicCredentialProviderConfig {
	if in == nil {
		return nil
	}
	out := new(DynamicCredentialProviderConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DynamicCredentialProviderConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MirrorConfig) DeepCopyInto(out *MirrorConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MirrorConfig.
func (in *MirrorConfig) DeepCopy() *MirrorConfig {
	if in == nil {
		return nil
	}
	out := new(MirrorConfig)
	in.DeepCopyInto(out)
	return out
}
