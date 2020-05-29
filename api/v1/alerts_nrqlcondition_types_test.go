// +build integration

package v1

import (
	"fmt"
	"reflect"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AlertsNrqlConditionSpec", func() {
	var condition AlertsNrqlConditionSpec

	BeforeEach(func() {

		condition = AlertsNrqlConditionSpec{
			Terms: []AlertsNrqlConditionTerm{
				{
					Operator:             alerts.NrqlConditionOperators.Above,
					Priority:             alerts.NrqlConditionPriorities.Critical,
					Threshold:            "5",
					ThresholdDuration:    60,
					ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
				},
			},
			Nrql: alerts.NrqlConditionQuery{
				Query:            "SELECT 1 FROM MyEvents",
				EvaluationOffset: 5,
			},
			Type:               "NRQL",
			Name:               "NRQL Condition",
			RunbookURL:         "http://test.com/runbook",
			ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
			ViolationTimeLimit: alerts.NrqlConditionViolationTimeLimits.OneHour,
			ID:                 777,
			ExpectedGroups:     2,
			IgnoreOverlap:      true,
			Enabled:            true,
			ExistingPolicyID:   "42",
		}
	})

	Describe("ToNrqlConditionInput", func() {
		It("converts AlertsNrqlConditionSpec object to NrqlConditionInput object, retaining field values", func() {
			conditionInput := condition.ToNrqlConditionInput()

			Expect(fmt.Sprint(reflect.TypeOf(conditionInput))).To(Equal("alerts.NrqlConditionInput"))

			// Expect(conditionInput.Type).To(Equal("NRQL"))
			Expect(conditionInput.Name).To(Equal("NRQL Condition"))
			Expect(conditionInput.RunbookURL).To(Equal("http://test.com/runbook"))
			Expect(string(*conditionInput.ValueFunction)).To(Equal(string(alerts.NrqlConditionValueFunctions.SingleValue)))
			//Expect(conditionInput.PolicyID).To(Equal(42))
			//Expect(conditionInput.ID).To(Equal(777))
			//Expect(conditionInput.ViolationCloseTimer).To(Equal(60))
			Expect(conditionInput.ViolationTimeLimit).To(Equal(alerts.NrqlConditionViolationTimeLimits.OneHour))
			//Expect(conditionInput.ExpectedGroups).To(Equal(2))
			//Expect(conditionInput.IgnoreOverlap).To(Equal(true))
			Expect(conditionInput.Enabled).To(Equal(true))

			apiTerm := conditionInput.Terms[0]

			Expect(fmt.Sprint(reflect.TypeOf(apiTerm))).To(Equal("alerts.NrqlConditionTerms"))

			//Expect(apiTerm.Duration).To(Equal(30))
			Expect(apiTerm.Operator).To(Equal(alerts.NrqlConditionOperators.Above))
			Expect(apiTerm.Priority).To(Equal(alerts.NrqlConditionPriorities.Critical))
			Expect(apiTerm.Threshold).To(Equal(float64(5)))
			Expect(apiTerm.ThresholdDuration).To(Equal(60))
			Expect(apiTerm.ThresholdOccurrences).To(Equal(alerts.ThresholdOccurrences.AtLeastOnce))

			apiQuery := conditionInput.Nrql

			Expect(fmt.Sprint(reflect.TypeOf(apiQuery))).To(Equal("alerts.NrqlConditionQuery"))

			Expect(apiQuery.Query).To(Equal("SELECT 1 FROM MyEvents"))
			//Expect(apiQuery.SinceValue).To(Equal("5"))
		})
	})
})
