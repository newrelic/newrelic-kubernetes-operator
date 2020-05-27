package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApmAlertConditionSpec defines the desired state of ApmAlertCondition
type ApmAlertConditionSpec struct {
	Terms               []AlertConditionTerm        `json:"terms,omitempty"`
	Type                string                      `json:"type,omitempty"` //TODO: add conditionType or pull from alerts package or make string
	Name                string                      `json:"name,omitempty"`
	RunbookURL          string                      `json:"runbook_url,omitempty"`
	Metric              string                      `json:"metric,omitempty"`
	UserDefined         alerts.ConditionUserDefined `json:"user_defined,omitempty"`
	Scope               string                      `json:"condition_scope,omitempty"`
	Entities            []string                    `json:"entities,omitempty"`
	GCMetric            string                      `json:"gc_metric,omitempty"`
	PolicyID            int                         `json:"-"`
	ID                  int                         `json:"id,omitempty"`
	ViolationCloseTimer int                         `json:"violation_close_timer,omitempty"`
	Enabled             bool                        `json:"enabled"`
	ExistingPolicyID    int                         `json:"existing_policy_id,omitempty"`
	APIKey              string                      `json:"api_key,omitempty"`
	APIKeySecret        NewRelicAPIKeySecret        `json:"api_key_secret,omitempty"`
	Region              string                      `json:"region,omitempty"`
}

// AlertConditionTerm represents the terms of a New Relic alert condition.
type AlertConditionTerm struct {
	Duration     string `json:"duration,omitempty"`
	Operator     string `json:"operator,omitempty"`
	Priority     string `json:"priority,omitempty"`
	Threshold    string `json:"threshold"`
	TimeFunction string `json:"time_function,omitempty"`
}

// ApmAlertConditionStatus defines the observed state of ApmAlertCondition
type ApmAlertConditionStatus struct {
	AppliedSpec *ApmAlertConditionSpec `json:"applied_spec"`
	ConditionID int                    `json:"condition_id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// ApmAlertCondition is the Schema for the apmalertconditions API
type ApmAlertCondition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApmAlertConditionSpec   `json:"spec,omitempty"`
	Status ApmAlertConditionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NrqlAlertConditionList contains a list of NrqlAlertCondition
type ApmAlertConditionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApmAlertCondition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApmAlertCondition{}, &ApmAlertConditionList{})
}

func (in ApmAlertConditionSpec) APICondition() alerts.Condition {
	jsonString, _ := json.Marshal(in)
	var APICondition alerts.Condition
	json.Unmarshal(jsonString, &APICondition) //nolint
	//APICondition.PolicyID = spec.ExistingPolicyId
	return APICondition
}
