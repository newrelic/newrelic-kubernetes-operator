package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlertsAPMConditionSpec defines the desired state of AlertsAPMCondition
type AlertsAPMConditionSpec struct {
	AlertsGenericConditionSpec `json:",inline"`
	AlertsAPMSpecificSpec      `json:",inline"`
}

type AlertsAPMSpecificSpec struct {
	Metric              string                      `json:"metric,omitempty"`
	UserDefined         alerts.ConditionUserDefined `json:"user_defined,omitempty"`
	Scope               string                      `json:"condition_scope,omitempty"`
	Entities            []string                    `json:"entities,omitempty"`
	GCMetric            string                      `json:"gc_metric,omitempty"`
	ViolationCloseTimer int                         `json:"violation_close_timer,omitempty"`
}

// AlertsAPMConditionStatus defines the observed state of AlertsAPMCondition
type AlertsAPMConditionStatus struct {
	AppliedSpec *AlertsAPMConditionSpec `json:"applied_spec"`
	ConditionID int                     `json:"condition_id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// AlertsAPMCondition is the Schema for the alertsapmconditions API
type AlertsAPMCondition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertsAPMConditionSpec   `json:"spec,omitempty"`
	Status AlertsAPMConditionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertsAPMConditionList contains a list of AlertsAPMCondition
type AlertsAPMConditionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertsAPMCondition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertsAPMCondition{}, &AlertsAPMConditionList{})
}

func (in AlertsAPMConditionSpec) APICondition() alerts.Condition {
	jsonString, _ := json.Marshal(in)
	var APICondition alerts.Condition
	json.Unmarshal(jsonString, &APICondition) //nolint

	// Set the condition terms from the APITerms slice, not the Terms slice.
	// This is due to the fact that the type definitions are different
	// between the REST API and Nerdgraph.
	for _, term := range in.AlertsGenericConditionSpec.APMTerms {
		jsonString, _ := json.Marshal(term)
		var APITerm alerts.ConditionTerm
		json.Unmarshal(jsonString, &APITerm) //nolint

		APICondition.Terms = append(APICondition.Terms, APITerm)
	}

	return APICondition
}
