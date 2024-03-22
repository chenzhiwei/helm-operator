/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HelmReleaseSpec defines the desired state of HelmRelease
type HelmReleaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Chart is Helm Chart object
	Chart HelmChart `json:"chart"`

	// Values is Helm Release values
	Values runtime.RawExtension `json:"values,omitempty"`

	// Foo is an example field of HelmRelease. Edit helmrelease_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// HelmChart is the Helm Chart
type HelmChart struct {
	// Helm Chart address
	Address string `json:"address"`

	// Credential is the secret that contains user/pass to download the chart
	Credential string `json:"credential,omitempty"`

	// skipTLSVerify skips the TLS verify when downloading the chart
	SkipTLSVerify bool `json:"skipTLSVerify,omitempty"`
}

// HelmReleaseStatus defines the observed state of HelmRelease
type HelmReleaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// HashedSpec is the hashcode of HelmReleaseSpec
	HashedSpec string `json:"hashedSpec,omitempty"`

	// Revision is the Helm Release revision
	Revision string `json:"revision,omitempty"`

	// Chart is chart that used for the Helm Release
	Chart string `json:"chart,omitempty"`

	// Updated is the Helm Release last updated time
	Updated string `json:"updated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HelmRelease is the Schema for the helmreleases API
type HelmRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmReleaseSpec   `json:"spec,omitempty"`
	Status HelmReleaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HelmReleaseList contains a list of HelmRelease
type HelmReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelmRelease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmRelease{}, &HelmReleaseList{})
}
