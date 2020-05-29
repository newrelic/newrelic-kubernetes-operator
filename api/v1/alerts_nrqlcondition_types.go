package v1

import (
	"strconv"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlertsNrqlConditionSpec defines the desired state of AlertsNrqlCondition
type AlertsNrqlConditionSpec struct {
	Enabled            bool                                   `json:"enabled"`
	IgnoreOverlap      bool                                   `json:"ignore_overlap,omitempty"`
	APIKey             string                                 `json:"api_key,omitempty"`
	APIKeySecret       NewRelicAPIKeySecret                   `json:"api_key_secret,omitempty"`
	AccountID          int                                    `json:"account_id,omitempty"`
	Description        string                                 `json:"description,omitempty"`
	ExistingPolicyID   string                                 `json:"existing_policy_id,omitempty"`
	ExpectedGroups     int                                    `json:"expected_groups,omitempty"`
	ID                 int                                    `json:"id,omitempty"`
	Name               string                                 `json:"name,omitempty"`
	Nrql               alerts.NrqlConditionQuery              `json:"nrql,omitempty"`
	PolicyID           int                                    `json:"-"`
	Region             string                                 `json:"region,omitempty"`
	RunbookURL         string                                 `json:"runbookUrl,omitempty"`
	Terms              []AlertsNrqlConditionTerm              `json:"terms,omitempty"`
	Type               alerts.NrqlConditionType               `json:"type,omitempty"`
	ValueFunction      *alerts.NrqlConditionValueFunction     `json:"valueFunction,omitempty"`
	ViolationTimeLimit alerts.NrqlConditionViolationTimeLimit `json:"violationTimeLimit,omitempty"`
}

// AlertsNrqlConditionQuery represents a NRQL query to use with a NRQL alert condition
type AlertsNrqlConditionQuery struct {
	Query      string `json:"query,omitempty"`
	SinceValue string `json:"since_value,omitempty"`
}

// AlertsNrqlConditionTerm represents the terms of a New Relic alert condition.
type AlertsNrqlConditionTerm struct {
	Operator             alerts.NrqlConditionOperator `json:"operator,omitempty"`
	Priority             alerts.NrqlConditionPriority `json:"priority,omitempty"`
	Threshold            string                       `json:"threshold,omitempty"`
	ThresholdDuration    int                          `json:"threshold_duration,omitempty"`
	ThresholdOccurrences alerts.ThresholdOccurrence   `json:"threshold_occurrences,omitempty"`
}

// AlertsNrqlConditionStatus defines the observed state of AlertsNrqlCondition
type AlertsNrqlConditionStatus struct {
	AppliedSpec *AlertsNrqlConditionSpec `json:"applied_spec"`
	ConditionID string                   `json:"condition_id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// AlertsNrqlCondition is the Schema for the alertsnrqlconditions API
type AlertsNrqlCondition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertsNrqlConditionSpec   `json:"spec,omitempty"`
	Status AlertsNrqlConditionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertsNrqlConditionList contains a list of AlertsNrqlCondition
type AlertsNrqlConditionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertsNrqlCondition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertsNrqlCondition{}, &AlertsNrqlConditionList{})
}

func (in AlertsNrqlConditionSpec) ToNrqlConditionInput() alerts.NrqlConditionInput {
	conditionInput := alerts.NrqlConditionInput{}
	conditionInput.Description = in.Description
	conditionInput.Enabled = in.Enabled
	conditionInput.Name = in.Name
	conditionInput.Nrql = in.Nrql
	conditionInput.RunbookURL = in.RunbookURL
	conditionInput.ViolationTimeLimit = in.ViolationTimeLimit

	if in.ValueFunction != nil {
		// f := alerts.NrqlConditionValueFunction(in.ValueFunction)
		conditionInput.ValueFunction = in.ValueFunction
	}

	// if in.BaselineDirection != nil {
	//      conditionInput.BaselineDirection = alerts.NrqlBaselineDirection(in.BaselineDirection)
	// }

	for _, term := range in.Terms {
		t := alerts.NrqlConditionTerms{}

		t.Operator = term.Operator
		t.Priority = term.Priority

		// When parsing a string
		f, err := strconv.ParseFloat(term.Threshold, 64)
		if err != nil {
			log.Error(err, "strconv.ParseFloat()", "threshold", term.Threshold)
		}

		t.Threshold = f
		t.ThresholdDuration = term.ThresholdDuration
		t.ThresholdOccurrences = term.ThresholdOccurrences

		conditionInput.Terms = append(conditionInput.Terms, t)
	}

	return conditionInput
}
