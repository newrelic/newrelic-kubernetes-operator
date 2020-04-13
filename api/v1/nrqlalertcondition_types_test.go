// +build integration

package v1

import (
	"fmt"
	"reflect"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("NrqlAlertConditionSpec", func() {
	var condition NrqlAlertConditionSpec

	BeforeEach(func() {
		condition = NrqlAlertConditionSpec{
			Terms: []AlertConditionTerm{
				{
					Duration:     resource.MustParse("30"),
					Operator:     "above",
					Priority:     "critical",
					Threshold:    resource.MustParse("5"),
					TimeFunction: "all",
				},
			},
			Nrql: NrqlQuery{
				Query:      "SELECT 1 FROM MyEvents",
				SinceValue: "5",
			},
			Type:                "NRQL",
			Name:                "NRQL Condition",
			RunbookURL:          "http://test.com/runbook",
			ValueFunction:       "max",
			ID:                  777,
			ViolationCloseTimer: 60,
			ExpectedGroups:      2,
			IgnoreOverlap:       true,
			Enabled:             true,
			ExistingPolicyID:    42,
		}
	})

	Describe("APICondition", func() {
		It("converts NrqlAlertConditionSpec object to NrqlCondition object from go client, retaining field values", func() {
			apiCondition := condition.APICondition()

			Expect(fmt.Sprint(reflect.TypeOf(apiCondition))).To(Equal("alerts.NrqlCondition"))

			Expect(apiCondition.Type).To(Equal("NRQL"))
			Expect(apiCondition.Name).To(Equal("NRQL Condition"))
			Expect(apiCondition.RunbookURL).To(Equal("http://test.com/runbook"))
			Expect(apiCondition.ValueFunction).To(Equal(alerts.ValueFunctionTypes.Max))
			//Expect(apiCondition.PolicyID).To(Equal(42))
			Expect(apiCondition.ID).To(Equal(777))
			Expect(apiCondition.ViolationCloseTimer).To(Equal(60))
			Expect(apiCondition.ExpectedGroups).To(Equal(2))
			Expect(apiCondition.IgnoreOverlap).To(Equal(true))
			Expect(apiCondition.Enabled).To(Equal(true))

			apiTerm := apiCondition.Terms[0]

			Expect(fmt.Sprint(reflect.TypeOf(apiTerm))).To(Equal("alerts.ConditionTerm"))

			Expect(apiTerm.Duration).To(Equal(30))
			Expect(apiTerm.Operator).To(Equal(alerts.OperatorTypes.Above))
			Expect(apiTerm.Priority).To(Equal(alerts.PriorityTypes.Critical))
			Expect(apiTerm.Threshold).To(Equal(float64(5)))
			Expect(apiTerm.TimeFunction).To(Equal(alerts.TimeFunctionTypes.All))

			apiQuery := apiCondition.Nrql

			Expect(fmt.Sprint(reflect.TypeOf(apiQuery))).To(Equal("alerts.NrqlQuery"))

			Expect(apiQuery.Query).To(Equal("SELECT 1 FROM MyEvents"))
			Expect(apiQuery.SinceValue).To(Equal("5"))
		})
	})
})
