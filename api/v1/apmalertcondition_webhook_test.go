// +build integration

package v1

import (
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ValidateCreate", func() {
	var (
		r ApmAlertCondition
		//alertsClient *interfacesfakes.FakeNewRelicAlertsClient
		//secret       *v1.Secret
	)

	Context("With a valid Condition", func() {
		r = ApmAlertCondition{
			ObjectMeta: v1.ObjectMeta{
				Name: "test apm condition",
			},
			Spec: ApmAlertConditionSpec{
				Terms: []NRAlertConditionTerm{
					{
						Duration:     "30",
						Operator:     "above",
						Priority:     "critical",
						Threshold:    "0.9",
						TimeFunction: "all",
					},
				},
				Type:             "apm_app_metric",
				Name:             "K8s generated apm alert condition",
				Metric:           "apdex",
				UserDefined:      alerts.ConditionUserDefined{},
				Scope:            "",
				Entities:         []string{"5950260"},
				Enabled:          true,
				ExistingPolicyID: 46286,
				APIKey:           "111222333",
				APIKeySecret:     NewRelicAPIKeySecret{},
				Region:           "staging",
			},
		}

		It("Should create the condition", func() {
			err := r.ValidateCreate()
			Expect(err).ToNot(HaveOccurred())

		})
	})
})
