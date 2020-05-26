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
			ExistingPolicyID:   42,
		}
	})

	Describe("APIConditionInput", func() {
		It("converts AlertsNrqlConditionSpec object to AlertsNrqlCondition object from go client, retaining field values", func() {
			apiCondition := condition.APIConditionInput()

			Expect(fmt.Sprint(reflect.TypeOf(apiCondition))).To(Equal("alerts.NrqlConditionInput"))

			// Expect(apiCondition.Type).To(Equal("NRQL"))
			Expect(apiCondition.Name).To(Equal("NRQL Condition"))
			Expect(apiCondition.RunbookURL).To(Equal("http://test.com/runbook"))
			Expect(string(*apiCondition.ValueFunction)).To(Equal(string(alerts.NrqlConditionValueFunctions.SingleValue)))
			//Expect(apiCondition.PolicyID).To(Equal(42))
			//Expect(apiCondition.ID).To(Equal(777))
			//Expect(apiCondition.ViolationCloseTimer).To(Equal(60))
			Expect(apiCondition.ViolationTimeLimit).To(Equal(alerts.NrqlConditionViolationTimeLimits.OneHour))
			//Expect(apiCondition.ExpectedGroups).To(Equal(2))
			//Expect(apiCondition.IgnoreOverlap).To(Equal(true))
			Expect(apiCondition.Enabled).To(Equal(true))

			apiTerm := apiCondition.Terms[0]

			Expect(fmt.Sprint(reflect.TypeOf(apiTerm))).To(Equal("alerts.NrqlConditionTerms"))

			//Expect(apiTerm.Duration).To(Equal(30))
			Expect(apiTerm.Operator).To(Equal(alerts.NrqlConditionOperators.Above))
			Expect(apiTerm.Priority).To(Equal(alerts.NrqlConditionPriorities.Critical))
			Expect(apiTerm.Threshold).To(Equal(float64(5)))
			Expect(apiTerm.ThresholdDuration).To(Equal(60))
			Expect(apiTerm.ThresholdOccurrences).To(Equal(alerts.ThresholdOccurrences.AtLeastOnce))

			apiQuery := apiCondition.Nrql

			Expect(fmt.Sprint(reflect.TypeOf(apiQuery))).To(Equal("alerts.NrqlConditionQuery"))

			Expect(apiQuery.Query).To(Equal("SELECT 1 FROM MyEvents"))
			//Expect(apiQuery.SinceValue).To(Equal("5"))
		})
	})
})
