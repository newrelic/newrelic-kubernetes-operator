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
	"hash/fnv"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PolicySpec defines the desired state of Policy
type PolicySpec struct {
	IncidentPreference string               `json:"incident_preference,omitempty"`
	Name               string               `json:"name"`
	APIKey             string               `json:"api_key,omitempty"`
	APIKeySecret       NewRelicAPIKeySecret `json:"api_key_secret,omitempty"`
	Region             string               `json:"region"`
	Conditions         []PolicyCondition    `json:"conditions,omitempty"`
}

//PolicyCondition defined the conditions contained within a a policy
type PolicyCondition struct {
	Name          string                 `json:"name"`
	Namespace     string                 `json:"namespace"`
	Spec          ConditionSpec          `json:"spec,omitempty"`
	NrqlAliasSpec NrqlAlertConditionSpec `json:"-"`
	ApmAliasSpec  ApmAlertConditionSpec  `json:"-"`
}

//type NrqlAlertConditionSpec = ConditionSpec
//type ApmAlertConditionSpec = ConditionSpec

type NrqlAliasSpec ConditionSpec
type ApmAliasSpec ConditionSpec

//ConditionSpec - Merged superset of Condition types
type ConditionSpec struct {
	GenericConditionSpec `json:",inline"`
	NrqlSpecificSpec     `json:",inline"`
	APMSpecificSpec      `json:",inline"`
}

// PolicyStatus defines the observed state of Policy
type PolicyStatus struct {
	AppliedSpec *PolicySpec `json:"applied_spec"`
	PolicyID    int         `json:"policy_id"`
}

// +kubebuilder:object:root=true

// Policy is the Schema for the policies API
type Policy struct {
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
	Items           []Policy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Policy{}, &PolicyList{})
}

func (in PolicySpec) APIPolicy() alerts.Policy {
	jsonString, _ := json.Marshal(in)
	var APIPolicy alerts.Policy
	json.Unmarshal(jsonString, &APIPolicy) //nolint

	//APICondition.PolicyID = spec.ExistingPolicyId

	return APIPolicy
}

func (p *PolicyCondition) SpecHash() uint32 {
	//remove api keys and condition from object to enable comparison minus inherited fields
	strippedPolicy := PolicyCondition{
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

func (p *PolicyCondition) GetNamespace() types.NamespacedName {
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

//GetConditionType - returns the string representative of the Condition type
func GetConditionType(condition PolicyCondition) string {
	if condition.Spec.Type == "NRQL" {
		return "NrqlAlertCondition"
	}
	return "ApmAlertCondition"

}

func (p *PolicyCondition) GenerateSpecFromNrqlConditionSpec(nrqlConditionSpec NrqlAlertConditionSpec) {
	jsonString, _ := json.Marshal(nrqlConditionSpec)
	json.Unmarshal(jsonString, &p.Spec) //nolint
}

func (p *PolicyCondition) GenerateSpecFromApmConditionSpec(apmConditionSpec ApmAlertConditionSpec) {
	jsonString, _ := json.Marshal(apmConditionSpec)
	json.Unmarshal(jsonString, &p.Spec) //nolint
}

func (p *PolicyCondition) ReturnNrqlConditionSpec() (nrqlAlertConditionSpec NrqlAlertConditionSpec) {
	jsonString, _ := json.Marshal(p.Spec)
	json.Unmarshal(jsonString, &nrqlAlertConditionSpec) //nolint
	return
}

func (p *PolicyCondition) ReturnApmConditionSpec() (apmAlertConditionSpec ApmAlertConditionSpec) {
	jsonString, _ := json.Marshal(p.Spec)
	json.Unmarshal(jsonString, &apmAlertConditionSpec) //nolint
	return
}
