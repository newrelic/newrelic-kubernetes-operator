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
		condition = AlertsNrqlConditionSpec{}
		condition.Enabled = true
		condition.ExistingPolicyID = "42"
		condition.Terms = []AlertsNrqlConditionTerm{
			{
				Operator:             alerts.AlertsNrqlConditionTermsOperatorTypes.ABOVE,
				Priority:             alerts.NrqlConditionPriorities.Critical,
				Threshold:            "5",
				ThresholdDuration:    60,
				ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
			},
		}
		condition.Nrql = alerts.NrqlConditionQuery{
			Query:            "SELECT 1 FROM MyEvents",
			EvaluationOffset: 5,
		}
		condition.Type = "NRQL"
		condition.Name = "NRQL Condition"
		condition.RunbookURL = "http://test.com/runbook"
		condition.ValueFunction = &alerts.NrqlConditionValueFunctions.SingleValue
		condition.ViolationTimeLimit = alerts.NrqlConditionViolationTimeLimits.OneHour
		condition.ID = 777
		condition.ExpectedGroups = 2
		condition.IgnoreOverlap = true
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

			Expect(fmt.Sprint(reflect.TypeOf(apiTerm))).To(Equal("alerts.NrqlConditionTerm"))

			//Expect(apiTerm.Duration).To(Equal(30))
			Expect(apiTerm.Operator).To(Equal(alerts.AlertsNrqlConditionTermsOperatorTypes.ABOVE))
			Expect(apiTerm.Priority).To(Equal(alerts.NrqlConditionPriorities.Critical))
			expectedThreshold := 5.0
			Expect(apiTerm.Threshold).To(Equal(&expectedThreshold))
			Expect(apiTerm.ThresholdDuration).To(Equal(60))
			Expect(apiTerm.ThresholdOccurrences).To(Equal(alerts.ThresholdOccurrences.AtLeastOnce))

			apiQuery := conditionInput.Nrql

			Expect(fmt.Sprint(reflect.TypeOf(apiQuery))).To(Equal("alerts.NrqlConditionQuery"))

			Expect(apiQuery.Query).To(Equal("SELECT 1 FROM MyEvents"))
			//Expect(apiQuery.SinceValue).To(Equal("5"))
		})
	})
})
