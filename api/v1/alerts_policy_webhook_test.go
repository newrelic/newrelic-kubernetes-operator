// +build integration

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("AlertsPolicy_webhooks", func() {
	Describe("validateCreate", func() {
		var (
			r            AlertsPolicy
			alertsClient *interfacesfakes.FakeNewRelicAlertsClient
			secret       *v1.Secret
		)

		BeforeEach(func() {
			k8Client = testk8sClient
			alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}
			fakeAlertFunc := func(string, string) (interfaces.NewRelicAlertsClient, error) {
				return alertsClient, nil
			}
			alertClientFunc = fakeAlertFunc

			r = AlertsPolicy{
				Spec: AlertsPolicySpec{
					Name:               "Test AlertsPolicy",
					IncidentPreference: "PER_POLICY",
					APIKey:             "api-key",
				},
			}

			// TODO: Make this a true integration test if possible
			alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
				return &alerts.Policy{
					ID: 42,
				}, nil
			}
		})

		Context("When given a valid API key", func() {
			It("should not return an error", func() {
				err := r.ValidateCreate()
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("When given an invalid API key", func() {
			It("should return an error", func() {
				r.Spec.APIKey = ""
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("either api_key or api_key_secret must be set"))
			})
		})

		Context("when given a valid API key in a secret", func() {
			It("should not return an error", func() {
				r.Spec.APIKey = ""
				r.Spec.APIKeySecret = NewRelicAPIKeySecret{
					Name:      "my-api-key-secret",
					Namespace: "my-namespace",
					KeyName:   "my-api-key",
				}
				secret = &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-api-key-secret",
						Namespace: "my-namespace",
					},
					Data: map[string][]byte{
						"my-api-key": []byte("data_here"),
					},
				}
				k8Client.Create(ctx, secret)
				err := r.ValidateCreate()
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				k8Client.Delete(ctx, secret)
			})
		})

		Context("when given a policy with an invalid incident_preference", func() {
			It("should reject the policy", func() {
				r.Spec.IncidentPreference = "totally bogus"
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("incident preference must be PER_POLICY, PER_CONDITION, or PER_CONDITION_AND_TARGET"))
			})
		})

		Context("when given a policy with duplicate conditions", func() {
			BeforeEach(func() {
				r.Spec.Conditions = []AlertsPolicyCondition{
					{
						Spec: AlertsNrqlConditionSpec{
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
							ExpectedGroups:     2,
							IgnoreOverlap:      true,
							Enabled:            true,
						},
					},
					{
						Spec: AlertsNrqlConditionSpec{
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
							ExpectedGroups:     2,
							IgnoreOverlap:      true,
							Enabled:            true,
						},
					},
				}
			})

			It("should reject the policy", func() {
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("duplicate conditions detected or hash collision"))
			})

			Context("and invalid API key and incident_preference", func() {
				It("should include all errors", func() {
					r.Spec.IncidentPreference = "totally bogus"
					r.Spec.APIKey = ""
					err := r.ValidateCreate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("either api_key or api_key_secret must be set"))
					Expect(err.Error()).To(ContainSubstring("duplicate conditions detected"))
					Expect(err.Error()).To(ContainSubstring("incident preference must be"))
				})
			})
		})
	})

	Describe("Default", func() {
		var (
			r AlertsPolicy
		)
		r = AlertsPolicy{
			Spec: AlertsPolicySpec{
				Name:               "Test AlertsPolicy",
				IncidentPreference: "PER_POLICY",
				APIKey:             "api-key",
				Conditions: []AlertsPolicyCondition{
					{
						Spec: AlertsNrqlConditionSpec{
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
							ExpectedGroups:     2,
							IgnoreOverlap:      true,
							Enabled:            true,
						},
					},
				},
			},
		}

		Context("when given a policy with no incident_preference set", func() {
			It("should set default value of PER_POLICY", func() {
				r.Spec.IncidentPreference = ""
				r.Default()
				Expect(r.Spec.IncidentPreference).To(Equal(string(alerts.AlertsIncidentPreferenceTypes.PER_POLICY)))
			})
		})

		Context("when given a policy with a lower case incident preference", func() {
			It("should upcase the incident preference", func() {
				r.Spec.IncidentPreference = "awesome-preference"
				r.Default()
				Expect(r.Spec.IncidentPreference).To(Equal("AWESOME-PREFERENCE"))
			})
		})
	})
})
