package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApmAlertConditionSpec defines the desired state of ApmAlertCondition
type ApmAlertConditionSpec struct {
	GenericConditionSpec `json:",inline"`
	APMSpecificSpec      `json:",inline"`
}

type APMSpecificSpec struct {
	Metric              string                      `json:"metric,omitempty"`
	UserDefined         alerts.ConditionUserDefined `json:"user_defined,omitempty"`
	Scope               string                      `json:"condition_scope,omitempty"`
	Entities            []string                    `json:"entities,omitempty"`
	GCMetric            string                      `json:"gc_metric,omitempty"`
	ViolationCloseTimer int                         `json:"violation_close_timer,omitempty"`
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

// ApmAlertConditionList contains a list of ApmAlertCondition
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

	return APICondition
}
