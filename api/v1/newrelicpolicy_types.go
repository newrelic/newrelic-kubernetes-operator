/*

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

// NewRelicPolicySpec defines the desired state of NewRelicPolicy
type NewRelicPolicySpec struct {
	IncidentPreference string `json:"incident_preference,omitempty"`
	Name               string `json:"name"`
	ID                 int    `json:"id"`
}

// NewRelicPolicyStatus defines the observed state of NewRelicPolicy
type NewRelicPolicyStatus struct {
	AppliedSpec *NewRelicPolicySpec `json:"applied_spec"`
	PolicyID    int                 `json:"policy_id"`
}

// +kubebuilder:object:root=true

// NewRelicPolicy is the Schema for the newrelicpolicies API
type NewRelicPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NewRelicPolicySpec   `json:"spec,omitempty"`
	Status NewRelicPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NewRelicPolicyList contains a list of NewRelicPolicy
type NewRelicPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NewRelicPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NewRelicPolicy{}, &NewRelicPolicyList{})
}
