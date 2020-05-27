// +build integration

package v1

import (
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("ValidateCreate", func() {
	var (
		r            ApmAlertCondition
		alertsClient *interfacesfakes.FakeNewRelicAlertsClient
		//secret       *v1.Secret
	)
	BeforeEach(func() {
		k8Client = testk8sClient
		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}
		fakeAlertFunc := func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}
		alertClientFunc = fakeAlertFunc
		r = ApmAlertCondition{
			ObjectMeta: v1.ObjectMeta{
				Name: "test apm condition",
			},
			Spec: ApmAlertConditionSpec{
				Terms: []AlertConditionTerm{
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

		alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
			return &alerts.Policy{
				ID: 46286,
			}, nil
		}
	})

	Context("With a valid Apm Condition", func() {

		It("Should create the apm condition", func() {
			err := r.ValidateCreate()
			Expect(err).ToNot(HaveOccurred())

		})
	})

	Context("With an invalid Type", func() {

		BeforeEach(func() {
			r.Spec.Type = "burritos"
		})

		It("Should reject the apm condition creation", func() {
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("burritos"))
		})
	})

	Context("With an invalid Metric", func() {

		BeforeEach(func() {
			r.Spec.Type = "moar burritos"
		})

		It("Should reject the apm condition creation", func() {
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("moar burritos"))
		})
	})
	Context("With an invalid Terms", func() {

		BeforeEach(func() {
			r.Spec.Terms[0].TimeFunction = "moar burritos"
			r.Spec.Terms[0].Priority = "moar tacos"
			r.Spec.Terms[0].Operator = "moar hamburgers"

		})

		It("Should reject the apm condition creation", func() {
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("moar burritos"))
			Expect(err.Error()).To(ContainSubstring("moar tacos"))
			Expect(err.Error()).To(ContainSubstring("moar hamburgers"))
		})
	})

	Context("With an invalid userDefined type", func() {

		BeforeEach(func() {
			r.Spec.UserDefined = alerts.ConditionUserDefined{
				Metric:        "Custom/foo",
				ValueFunction: "invalid type",
			}
		})

		It("Should reject the apm condition creation", func() {
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid type"))
		})
	})
})
