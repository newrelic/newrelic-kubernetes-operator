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

// AlertsPolicySpec defines the desired state of AlertsPolicy
type AlertsPolicySpec struct {
	IncidentPreference string                  `json:"incidentPreference,omitempty"`
	Name               string                  `json:"name"`
	Region             string                  `json:"region"`
	Conditions         []AlertsPolicyCondition `json:"conditions,omitempty"`
	APIKey             string                  `json:"api_key,omitempty"`
	APIKeySecret       NewRelicAPIKeySecret    `json:"api_key_secret,omitempty"`
	AccountID          int                     `json:"account_id,omitempty"`
}

//AlertsPolicyCondition defined the conditions contained within an AlertsPolicy
type AlertsPolicyCondition struct {
	Name      string                    `json:"name,omitempty"`
	Namespace string                    `json:"namespace,omitempty"`
	Spec      AlertsPolicyConditionSpec `json:"spec,omitempty"`
}

type AlertsPolicyConditionSpec struct {
	AlertsGenericConditionSpec `json:",inline"`
	AlertsNrqlSpecificSpec     `json:",inline"`
	AlertsAPMSpecificSpec      `json:",inline"`
}

// AlertsPolicyStatus defines the observed state of AlertsPolicy
type AlertsPolicyStatus struct {
	AppliedSpec *AlertsPolicySpec `json:"applied_spec"`
	PolicyID    string            `json:"policy_id"`
}

// +kubebuilder:object:root=true

// AlertsPolicy is the Schema for the policies API
type AlertsPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertsPolicySpec   `json:"spec,omitempty"`
	Status AlertsPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertsPolicyList contains a list of AlertsPolicy
type AlertsPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertsPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertsPolicy{}, &AlertsPolicyList{})
}

func (in AlertsPolicySpec) ToAlertsPolicy() alerts.AlertsPolicy {
	jsonString, _ := json.Marshal(in)
	var result alerts.AlertsPolicy
	json.Unmarshal(jsonString, &result) //nolint

	return result
}

func (in AlertsPolicySpec) ToAlertsPolicyUpdateInput() alerts.AlertsPolicyUpdateInput {
	jsonString, _ := json.Marshal(in)
	var result alerts.AlertsPolicyUpdateInput
	json.Unmarshal(jsonString, &result) //nolint

	return result
}

func (in AlertsPolicySpec) ToAlertsPolicyInput() alerts.AlertsPolicyInput {
	jsonString, _ := json.Marshal(in)
	var result alerts.AlertsPolicyInput
	json.Unmarshal(jsonString, &result) //nolint

	return result
}

func (p *AlertsPolicyCondition) SpecHash() uint32 {
	//remove api keys and condition from object to enable comparison minus inherited fields
	strippedAlertsPolicy := AlertsPolicyCondition{
		Spec: p.Spec,
	}
	strippedAlertsPolicy.Spec.APIKeySecret = NewRelicAPIKeySecret{}
	strippedAlertsPolicy.Spec.APIKey = ""
	strippedAlertsPolicy.Spec.Region = ""
	strippedAlertsPolicy.Spec.ExistingPolicyID = ""
	conditionTemplateSpecHasher := fnv.New32a()
	DeepHashObject(conditionTemplateSpecHasher, strippedAlertsPolicy)
	return conditionTemplateSpecHasher.Sum32()
}

func (p *AlertsPolicyCondition) GetNamespace() types.NamespacedName {
	return types.NamespacedName{
		Namespace: p.Namespace,
		Name:      p.Name,
	}
}

//Equals - comparator function to check for equality
func (in AlertsPolicySpec) Equals(policyToCompare AlertsPolicySpec) bool {
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

//GetAlertsConditionType - returns the string representative of the Condition type
func GetAlertsConditionType(condition AlertsPolicyCondition) string {
	if condition.Spec.Type == "NRQL" {
		return "AlertsNrqlCondition"
	}
	return "ApmAlertCondition"

}

func (p *AlertsPolicyCondition) GenerateSpecFromNrqlConditionSpec(nrqlConditionSpec AlertsNrqlConditionSpec) {
	jsonString, _ := json.Marshal(nrqlConditionSpec)
	json.Unmarshal(jsonString, &p.Spec) //nolint
}

func (p *AlertsPolicyCondition) GenerateSpecFromApmConditionSpec(apmConditionSpec ApmAlertConditionSpec) {
	jsonString, _ := json.Marshal(apmConditionSpec)
	json.Unmarshal(jsonString, &p.Spec) //nolint
}

func (p *AlertsPolicyCondition) ReturnNrqlConditionSpec() (nrqlConditionSpec AlertsNrqlConditionSpec) {
	jsonString, _ := json.Marshal(p.Spec)
	json.Unmarshal(jsonString, &nrqlConditionSpec) //nolint
	return
}

func (p *AlertsPolicyCondition) ReturnApmConditionSpec() (apmAlertConditionSpec ApmAlertConditionSpec) {
	jsonString, _ := json.Marshal(p.Spec)
	json.Unmarshal(jsonString, &apmAlertConditionSpec) //nolint
	return
}
