/*
Copyright 2021.

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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Resource is the resource indentifier
type Resource struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// HelmDogSpec defines the desired state of HelmDog
type HelmDogSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Resources is the resources that to be deleted when uninstall a helm chart
	Resources []Resource `json:"resources"`
}

// HelmDogStatus defines the observed state of HelmDog
type HelmDogStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Resources is the current resources in HelmDog
	Resources []Resource `json:"resources"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HelmDog is the Schema for the helmdogs API
type HelmDog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmDogSpec   `json:"spec,omitempty"`
	Status HelmDogStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HelmDogList contains a list of HelmDog
type HelmDogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelmDog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmDog{}, &HelmDogList{})
}
