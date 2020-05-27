// +build integration

package v1

import (
	"fmt"
	"reflect"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ApmAlertConditionSpec", func() {
	var condition ApmAlertConditionSpec

	BeforeEach(func() {
		condition = ApmAlertConditionSpec{
			Terms: []AlertConditionTerm{
				{
					Duration:     "30",
					Operator:     "above",
					Priority:     "critical",
					Threshold:    "1.5",
					TimeFunction: "all",
				},
			},
			Type:       "apm_app_metric",
			Name:       "APM Condition",
			RunbookURL: "http://test.com/runbook",
			Metric:     "apdex",
			Entities:   []string{"333"},
			UserDefined: alerts.ConditionUserDefined{
				Metric:        "Custom/foo",
				ValueFunction: "average",
			},
			Scope:               "application",
			GCMetric:            "",
			PolicyID:            0,
			ID:                  888,
			ViolationCloseTimer: 60,
			Enabled:             true,
			ExistingPolicyID:    42,
		}
	})

	Describe("APICondition", func() {
		It("converts ApmAlertConditionSpec object to Condition object from go client, retaining field values", func() {
			apiCondition := condition.APICondition()

			Expect(fmt.Sprint(reflect.TypeOf(apiCondition))).To(Equal("alerts.Condition"))

			Expect(apiCondition.Type).To(Equal(alerts.ConditionTypes.APMApplicationMetric))
			Expect(apiCondition.Name).To(Equal("APM Condition"))
			Expect(apiCondition.RunbookURL).To(Equal("http://test.com/runbook"))
			Expect(apiCondition.Metric).To(Equal(alerts.MetricTypes.Apdex))
			Expect(apiCondition.Entities[0]).To(Equal("333"))
			Expect(apiCondition.Scope).To(Equal("application"))
			Expect(apiCondition.GCMetric).To(Equal(""))

			Expect(apiCondition.ID).To(Equal(888))
			Expect(apiCondition.ViolationCloseTimer).To(Equal(60))
			Expect(apiCondition.Enabled).To(Equal(true))

			apiTerm := apiCondition.Terms[0]

			Expect(fmt.Sprint(reflect.TypeOf(apiTerm))).To(Equal("alerts.ConditionTerm"))

			Expect(apiTerm.Duration).To(Equal(30))
			Expect(apiTerm.Operator).To(Equal(alerts.OperatorTypes.Above))
			Expect(apiTerm.Priority).To(Equal(alerts.PriorityTypes.Critical))
			Expect(apiTerm.Threshold).To(Equal(float64(1.5)))
			Expect(apiTerm.TimeFunction).To(Equal(alerts.TimeFunctionTypes.All))

			userDefinedCondition := apiCondition.UserDefined
			Expect(fmt.Sprint(reflect.TypeOf(userDefinedCondition))).To(Equal("alerts.ConditionUserDefined"))

			Expect(userDefinedCondition.Metric).To(Equal("Custom/foo"))
			Expect(userDefinedCondition.ValueFunction).To(Equal(alerts.ValueFunctionTypes.Average))
		})
	})
})
