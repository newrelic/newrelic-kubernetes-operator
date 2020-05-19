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
	"encoding/json"
	"hash"
	"hash/fnv"

	"github.com/davecgh/go-spew/spew"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PolicySpec defines the desired state of Policy
type PolicySpec struct {
	APIKey             string                  `json:"api_key,omitempty"`
	APIKeySecret       NewRelicAPIKeySecret    `json:"api_key_secret,omitempty"`
	Conditions         []PolicyConditionSchema `json:"conditions,omitempty"`
	IncidentPreference string                  `json:"incident_preference,omitempty"`
	Name               string                  `json:"name"`
	Region             string                  `json:"region"`
}

//PolicyConditionSchema defined the conditions contained within a a policy
type PolicyConditionSchema struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Spec      NrqlConditionSpec `json:"spec,omitempty"`
}

// PolicyStatus defines the observed state of Policy
type PolicyStatus struct {
	AppliedSpec *PolicySpec `json:"applied_spec"`
	PolicyID    int         `json:"policy_id"`
}

// +kubebuilder:object:root=true

// PolicySchema is the Schema for the policies API
type PolicySchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicySpec   `json:"spec,omitempty"`
	Status PolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PolicyList contains a list of Policy
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PolicySchema `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PolicySchema{}, &PolicyList{})
}

func (spec PolicySpec) ToPolicy() alerts.Policy {
	jsonString, _ := json.Marshal(spec)
	var policy alerts.Policy
	json.Unmarshal(jsonString, &policy) //nolint

	return policy
}

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func DeepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	printer.Fprintf(hasher, "%#v", objectToWrite)
}

func (p *PolicyConditionSchema) SpecHash() uint32 {
	//remove api keys and condition from object to enable comparison minus inherited fields
	strippedPolicy := NrqlConditionSchema{
		Spec: p.Spec,
	}
	strippedPolicy.Spec.APIKeySecret = NewRelicAPIKeySecret{}
	strippedPolicy.Spec.APIKey = ""
	strippedPolicy.Spec.Region = ""
	strippedPolicy.Spec.ExistingPolicyID = 0
	conditionTemplateSpecHasher := fnv.New32a()
	DeepHashObject(conditionTemplateSpecHasher, strippedPolicy)
	return conditionTemplateSpecHasher.Sum32()
}

func (p *PolicyConditionSchema) GetNamespace() types.NamespacedName {
	return types.NamespacedName{
		Namespace: p.Namespace,
		Name:      p.Name,
	}
}

//Equals - comparator function to check for equality
func (in PolicySpec) Equals(policyToCompare PolicySpec) bool {
	if in.IncidentPreference != policyToCompare.IncidentPreference {
		return false
	}
	if in.Name != policyToCompare.Name {
		return false
	}
	if in.APIKey != policyToCompare.APIKey {
		return false
	}
	if in.Region != policyToCompare.Region {
		return false
	}
	if in.APIKeySecret != policyToCompare.APIKeySecret {
		return false
	}
	if len(in.Conditions) != len(policyToCompare.Conditions) {
		return false
	}

	checkedHashes := make(map[uint32]bool)

	for _, condition := range in.Conditions {
		checkedHashes[condition.SpecHash()] = true
	}

	for _, conditionToCompare := range policyToCompare.Conditions {
		if _, ok := checkedHashes[conditionToCompare.SpecHash()]; !ok {
			return false
		}
	}
	return true
}
