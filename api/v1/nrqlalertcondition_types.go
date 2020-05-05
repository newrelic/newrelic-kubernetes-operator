package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NrqlAlertConditionSpec defines the desired state of NrqlAlertCondition
type NrqlAlertConditionSpec struct {
	Terms               []AlertConditionTerm `json:"terms,omitempty"`
	Nrql                NrqlQuery            `json:"nrql,omitempty"`
	Type                string               `json:"type,omitempty"`
	Name                string               `json:"name,omitempty"`
	RunbookURL          string               `json:"runbook_url,omitempty"`
	ValueFunction       string               `json:"value_function,omitempty"`
	PolicyID            int                  `json:"-"`
	ID                  int                  `json:"id,omitempty"`
	ViolationCloseTimer int                  `json:"violation_time_limit_seconds,omitempty"`
	ExpectedGroups      int                  `json:"expected_groups,omitempty"`
	IgnoreOverlap       bool                 `json:"ignore_overlap,omitempty"`
	Enabled             bool                 `json:"enabled"`
	ExistingPolicyID    int                  `json:"existing_policy_id,omitempty"`
	APIKey              string               `json:"api_key,omitempty"`
	APIKeySecret        NewRelicAPIKeySecret `json:"api_key_secret,omitempty"`
	Region              string               `json:"region,omitempty"`
}

// NrqlQuery represents a NRQL query to use with a NRQL alert condition
type NrqlQuery struct {
	Query      string `json:"query,omitempty"`
	SinceValue string `json:"since_value,omitempty"`
}

// AlertConditionTerm represents the terms of a New Relic alert condition.
type AlertConditionTerm struct {
	Duration     resource.Quantity `json:"duration,omitempty"`
	Operator     string            `json:"operator,omitempty"`
	Priority     string            `json:"priority,omitempty"`
	Threshold    resource.Quantity `json:"threshold"`
	TimeFunction string            `json:"time_function,omitempty"`
}

// NrqlAlertConditionStatus defines the observed state of NrqlAlertCondition
type NrqlAlertConditionStatus struct {
	AppliedSpec *NrqlAlertConditionSpec `json:"applied_spec"`
	ConditionID int                     `json:"condition_id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// NrqlAlertCondition is the Schema for the nrqlalertconditions API
type NrqlAlertCondition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NrqlAlertConditionSpec   `json:"spec,omitempty"`
	Status NrqlAlertConditionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NrqlAlertConditionList contains a list of NrqlAlertCondition
type NrqlAlertConditionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NrqlAlertCondition `json:"items"`
}

type NewRelicAPIKeySecret struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	KeyName   string `json:"key_name,omitempty"`
}

func init() {
	SchemeBuilder.Register(&NrqlAlertCondition{}, &NrqlAlertConditionList{})
}

func (in NrqlAlertConditionSpec) APICondition() alerts.NrqlCondition {
	jsonString, _ := json.Marshal(in)
	var APICondition alerts.NrqlCondition
	json.Unmarshal(jsonString, &APICondition) //nolint

	//APICondition.PolicyID = spec.ExistingPolicyId

	return APICondition
}
