package v1

import (
	"fmt"
	"strconv"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NrqlAlertConditionSpec defines the desired state of NrqlAlertCondition
type NrqlAlertConditionSpec struct {
	APIKey             string                                 `json:"api_key,omitempty"`
	APIKeySecret       NewRelicAPIKeySecret                   `json:"api_key_secret,omitempty"`
	AccountID          int                                    `json:"account_id,omitempty"`
	Description        string                                 `json:"description,omitempty"`
	Enabled            bool                                   `json:"enabled"`
	ExistingPolicyID   int                                    `json:"existing_policy_id,omitempty"`
	ExpectedGroups     int                                    `json:"expected_groups,omitempty"`
	ID                 int                                    `json:"id,omitempty"`
	IgnoreOverlap      bool                                   `json:"ignore_overlap,omitempty"`
	Name               string                                 `json:"name,omitempty"`
	Nrql               alerts.NrqlConditionQuery              `json:"nrql,omitempty"`
	PolicyID           int                                    `json:"-"`
	Region             string                                 `json:"region,omitempty"`
	RunbookURL         string                                 `json:"runbookUrl,omitempty"`
	Terms              []AlertConditionTerm                   `json:"terms,omitempty"`
	Type               alerts.NrqlConditionType               `json:"type,omitempty"`
	ValueFunction      *alerts.NrqlConditionValueFunction     `json:"valueFunction,omitempty"`
	ViolationTimeLimit alerts.NrqlConditionViolationTimeLimit `json:"violationTimeLimit,omitempty"`
}

// NrqlQuery represents a NRQL query to use with a NRQL alert condition
// TODO replace with alerts.NrqlConditionQuery
type NrqlQuery struct {
	Query      string `json:"query,omitempty"`
	SinceValue string `json:"since_value,omitempty"`
}

// AlertConditionTerm represents the terms of a New Relic alert condition.
type AlertConditionTerm struct {
	Operator             alerts.NrqlConditionOperator `json:"operator,omitempty"`
	Priority             alerts.NrqlConditionPriority `json:"priority,omitempty"`
	Threshold            string                       `json:"threshold,omitempty"`
	ThresholdDuration    int                          `json:"threshold_duration,omitempty"`
	ThresholdOccurrences alerts.ThresholdOccurrence   `json:"threshold_occurrences,omitempty"`
	TimeFunction         string                       `json:"time_function,omitempty"`
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

func (in NrqlAlertConditionSpec) APIConditionBase() alerts.NrqlConditionBase {
	conditionBase := alerts.NrqlConditionBase{
		Description:        in.Description,
		Enabled:            in.Enabled,
		Name:               in.Name,
		Nrql:               in.Nrql,
		RunbookURL:         in.RunbookURL,
		ViolationTimeLimit: in.ViolationTimeLimit,
	}

	for _, term := range in.Terms {
		t := alerts.NrqlConditionTerms{}

		t.Operator = term.Operator
		t.Priority = term.Priority
		// TODO handle the float here

		// When parsing a string
		f, err := strconv.ParseFloat(term.Threshold, 64)
		if err != nil {
			log.Error(err, "strconv.ParseFloat()", "threshold", term.Threshold)
		}

		// When parsing a resource.Quantity
		// j, err := json.Marshal(term.Threshold)
		// if err != nil {
		// 	log.Error(err, "json marshal", "threshold", term.Threshold)
		// }

		// var f float64
		// err = json.Unmarshal(j, &f) //nolint
		// if err != nil {
		// 	log.Error(err, "json unmarshal", "j", string(j))
		// }

		t.Threshold = f
		t.ThresholdDuration = term.ThresholdDuration
		t.ThresholdOccurrences = term.ThresholdOccurrences

		fmt.Printf("t: %+v\n\n\n", t)

		conditionBase.Terms = append(conditionBase.Terms, t)
	}

	return conditionBase
}

// APINrqlAlertCondition is the same as APIConditionBase except that additional fields on the struct are also reviewed.
func (in NrqlAlertConditionSpec) APINrqlAlertCondition() alerts.NrqlAlertCondition {
	condition := alerts.NrqlAlertCondition{}

	condition.Description = in.Description
	condition.Enabled = in.Enabled
	condition.Name = in.Name
	condition.Nrql = in.Nrql
	condition.RunbookURL = in.RunbookURL
	condition.ViolationTimeLimit = in.ViolationTimeLimit

	for _, term := range in.Terms {
		t := alerts.NrqlConditionTerms{}

		t.Operator = term.Operator
		t.Priority = term.Priority
		// TODO handle the float here

		// When parsing a string
		f, err := strconv.ParseFloat(term.Threshold, 64)
		if err != nil {
			log.Error(err, "strconv.ParseFloat()", "threshold", term.Threshold)
		}

		t.Threshold = f
		t.ThresholdDuration = term.ThresholdDuration
		t.ThresholdOccurrences = term.ThresholdOccurrences

		fmt.Printf("t: %+v\n\n\n", t)

		condition.Terms = append(condition.Terms, t)
	}

	condition.ID = strconv.Itoa(in.ID)
	condition.ValueFunction = in.ValueFunction

	return condition
}
