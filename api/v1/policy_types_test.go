// +build integration

package v1

import (
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Equals", func() {
	var (
		p               PolicySpec
		policyToCompare PolicySpec
		output          bool
		condition       PolicyCondition
	)

	BeforeEach(func() {
		condition = PolicyCondition{
			Name:      "policy-name",
			Namespace: "default",
			Spec: ConditionSpec{
				Terms: []AlertConditionTerm{
					{
						Duration:     "30",
						Operator:     "above",
						Priority:     "critical",
						Threshold:    "5",
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
			},
		}

		p = PolicySpec{
			IncidentPreference: "PER_POLICY",
			Name:               "test-policy",
			APIKey:             "112233",
			APIKeySecret: NewRelicAPIKeySecret{
				Name:      "secret",
				Namespace: "default",
				KeyName:   "api-key",
			},
			Region:     "us",
			Conditions: []PolicyCondition{condition},
		}

		policyToCompare = PolicySpec{
			IncidentPreference: "PER_POLICY",
			Name:               "test-policy",
			APIKey:             "112233",
			APIKeySecret: NewRelicAPIKeySecret{
				Name:      "secret",
				Namespace: "default",
				KeyName:   "api-key",
			},
			Region:     "us",
			Conditions: []PolicyCondition{condition},
		}
	})

	Context("When equal", func() {
		It("should return true", func() {
			output = p.Equals(policyToCompare)
			Expect(output).To(BeTrue())
		})
	})

	Context("When condition hash matches", func() {
		It("should return true", func() {
			output = p.Equals(policyToCompare)
			Expect(output).To(BeTrue())
		})
	})

	Context("When condition hash matches but k8s condition name doesn't", func() {
		It("should return true", func() {
			p.Conditions = []PolicyCondition{
				{
					Name:      "",
					Namespace: "",
					Spec: ConditionSpec{
						Terms: []AlertConditionTerm{
							{
								Duration:     "30",
								Operator:     "above",
								Priority:     "critical",
								Threshold:    "5",
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
					},
				},
			}
			output = p.Equals(policyToCompare)
			Expect(output).To(BeTrue())
		})
	})

	Context("When condition hash doesn't match matches but name does", func() {
		It("should return false", func() {
			p.Conditions = []PolicyCondition{
				{
					Name:      "policy-name",
					Namespace: "default",
					Spec: ConditionSpec{
						Name: "test condition 222",
					},
				},
			}
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When one condition hash doesn't match but the other does", func() {
		It("should return false", func() {
			p.Conditions = []PolicyCondition{
				{
					Spec: ConditionSpec{
						Name: "test condition",
					},
				},
				{
					Spec: ConditionSpec{
						Name: "test condition 2",
					},
				},
			}
			policyToCompare.Conditions = []PolicyCondition{
				{
					Spec: ConditionSpec{
						Name: "test condition",
					},
				},
				{
					Spec: ConditionSpec{
						Name: "test condition is awesome",
					},
				},
			}
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When different number of conditions exist", func() {
		It("should return false", func() {
			p.Conditions = []PolicyCondition{
				{
					Spec: ConditionSpec{
						Name: "test condition",
					},
				},
				{
					Spec: ConditionSpec{
						Name: "test condition 2",
					},
				},
			}
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When Incident preference doesn't match", func() {
		It("should return false", func() {
			p.IncidentPreference = "PER_CONDITION"
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When region doesn't match", func() {
		It("should return false", func() {
			p.Region = "eu"
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When APIKeysecret doesn't match", func() {
		It("should return false", func() {
			p.APIKeySecret = NewRelicAPIKeySecret{
				Name:      "new secret",
				Namespace: "default",
				KeyName:   "api-key",
			}
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})
})

var _ = Describe("GetNrqlConditionSpec", func() {
	Context("With a NRQL condition type PolicyCondition", func() {
		var condition PolicyCondition
		condition = PolicyCondition{
			Name:      "my-policy",
			Namespace: "default",
			Spec: ConditionSpec{
				Terms: []AlertConditionTerm{
					{
						Duration:     "30",
						Operator:     "above",
						Priority:     "critical",
						Threshold:    "5",
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
			},
		}
		It("Should return a matching NrqlConditionSpec", func() {
			nrqlConditionSpec := condition.ReturnNrqlConditionSpec()
			Expect(nrqlConditionSpec.Type).To(Equal("NRQL"))
			Expect(nrqlConditionSpec.Name).To(Equal("NRQL Condition"))
			Expect(nrqlConditionSpec.RunbookURL).To(Equal("http://test.com/runbook"))
			Expect(nrqlConditionSpec.ValueFunction).To(Equal("max"))
			Expect(nrqlConditionSpec.ID).To(Equal(777))
			Expect(nrqlConditionSpec.ViolationCloseTimer).To(Equal(60))
			Expect(nrqlConditionSpec.ExpectedGroups).To(Equal(2))
			Expect(nrqlConditionSpec.IgnoreOverlap).To(BeTrue())
			Expect(nrqlConditionSpec.Enabled).To(BeTrue())
			Expect(nrqlConditionSpec.ExistingPolicyID).To(Equal(42))
			expectedAlertConditionTerm := AlertConditionTerm{
				Duration:     "30",
				Operator:     "above",
				Priority:     "critical",
				Threshold:    "5",
				TimeFunction: "all",
			}
			Expect(nrqlConditionSpec.Terms[0]).To(Equal(expectedAlertConditionTerm))
			expectedNrql := NrqlQuery{
				Query:      "SELECT 1 FROM MyEvents",
				SinceValue: "5",
			}
			Expect(nrqlConditionSpec.Nrql).To(Equal(expectedNrql))
		})
	})
})

var _ = Describe("GetApmConditionSpec", func() {
	Context("With a NRQL condition type PolicyCondition", func() {
		var condition PolicyCondition
		condition = PolicyCondition{
			Name:      "my-policy",
			Namespace: "default",
			Spec: ConditionSpec{
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
			},
		}
		It("Should return a matching ApmConditionSpec", func() {
			apmConditionSpec := condition.ReturnApmConditionSpec()
			Expect(apmConditionSpec.Type).To(Equal("apm_app_metric"))
			Expect(apmConditionSpec.Name).To(Equal("APM Condition"))
			Expect(apmConditionSpec.RunbookURL).To(Equal("http://test.com/runbook"))
			Expect(apmConditionSpec.ID).To(Equal(888))
			Expect(apmConditionSpec.ViolationCloseTimer).To(Equal(60))
			Expect(apmConditionSpec.Enabled).To(BeTrue())
			Expect(apmConditionSpec.ExistingPolicyID).To(Equal(42))
			expectedAlertConditionTerm := AlertConditionTerm{
				Duration:     "30",
				Operator:     "above",
				Priority:     "critical",
				Threshold:    "1.5",
				TimeFunction: "all",
			}
			Expect(apmConditionSpec.Terms[0]).To(Equal(expectedAlertConditionTerm))

			Expect(apmConditionSpec.Entities[0]).To(Equal("333"))
			Expect(apmConditionSpec.Scope).To(Equal("application"))
			expectedUserDefined := alerts.ConditionUserDefined{
				Metric:        "Custom/foo",
				ValueFunction: "average",
			}
			Expect(apmConditionSpec.UserDefined).To(Equal(expectedUserDefined))
			Expect(apmConditionSpec.Metric).To(Equal("apdex"))
		})
	})
})
