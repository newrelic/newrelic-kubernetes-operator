// +build integration

package v1

import (
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Equals", func() {
	var (
		p               AlertsPolicySpec
		policyToCompare AlertsPolicySpec
		output          bool
		condition       AlertsPolicyCondition
	)

	BeforeEach(func() {
		spec := AlertsPolicyConditionSpec{}
		spec.Terms = []AlertsNrqlConditionTerm{
			{
				Operator:             alerts.AlertsNrqlConditionTermsOperatorTypes.ABOVE,
				Priority:             alerts.NrqlConditionPriorities.Critical,
				Threshold:            "5",
				ThresholdDuration:    60,
				ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
			},
		}
		spec.Nrql = alerts.NrqlConditionQuery{
			Query:            "SELECT 1 FROM MyEvents",
			EvaluationOffset: 5,
		}
		spec.Type = "NRQL"
		spec.Name = "NRQL Condition"
		spec.RunbookURL = "http://test.com/runbook"
		spec.ValueFunction = &alerts.NrqlConditionValueFunctions.SingleValue
		spec.ID = 777
		spec.ViolationTimeLimit = alerts.NrqlConditionViolationTimeLimits.OneHour
		spec.ExpectedGroups = 2
		spec.IgnoreOverlap = true
		spec.Enabled = true
		spec.ExistingPolicyID = "42"

		condition = AlertsPolicyCondition{
			Name:      "policy-name",
			Namespace: "default",
			Spec:      spec,
		}

		p = AlertsPolicySpec{
			IncidentPreference: "PER_POLICY",
			Name:               "test-policy",
			APIKey:             "112233",
			APIKeySecret: NewRelicAPIKeySecret{
				Name:      "secret",
				Namespace: "default",
				KeyName:   "api-key",
			},
			Region:     "us",
			Conditions: []AlertsPolicyCondition{condition},
		}

		policyToCompare = AlertsPolicySpec{
			IncidentPreference: "PER_POLICY",
			Name:               "test-policy",
			APIKey:             "112233",
			APIKeySecret: NewRelicAPIKeySecret{
				Name:      "secret",
				Namespace: "default",
				KeyName:   "api-key",
			},
			Region:     "us",
			Conditions: []AlertsPolicyCondition{condition},
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
			spec := AlertsPolicyConditionSpec{}

			spec.Terms = []AlertsNrqlConditionTerm{
				{
					Operator:             alerts.AlertsNrqlConditionTermsOperatorTypes.ABOVE,
					Priority:             alerts.NrqlConditionPriorities.Critical,
					Threshold:            "5",
					ThresholdDuration:    60,
					ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
				},
			}
			spec.Nrql = alerts.NrqlConditionQuery{
				Query:            "SELECT 1 FROM MyEvents",
				EvaluationOffset: 5,
			}
			spec.Type = "NRQL"
			spec.Name = "NRQL Condition"
			spec.RunbookURL = "http://test.com/runbook"
			spec.ValueFunction = &alerts.NrqlConditionValueFunctions.SingleValue
			spec.ID = 777
			spec.ViolationTimeLimit = alerts.NrqlConditionViolationTimeLimits.OneHour
			spec.ExpectedGroups = 2
			spec.IgnoreOverlap = true
			spec.Enabled = true
			spec.ExistingPolicyID = "42"

			p.Conditions = []AlertsPolicyCondition{
				{
					Name:      "",
					Namespace: "",
					Spec:      spec,
				},
			}
			output = p.Equals(policyToCompare)
			Expect(output).To(BeTrue())
		})
	})

	Context("When condition hash doesn't match matches but name does", func() {
		It("should return false", func() {
			spec := AlertsPolicyConditionSpec{}
			spec.Name = "test condition 222"

			p.Conditions = []AlertsPolicyCondition{
				{
					Name:      "policy-name",
					Namespace: "default",
					Spec:      spec,
				},
			}

			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When one condition hash doesn't match but the other does", func() {
		It("should return false", func() {
			spec1 := AlertsPolicyConditionSpec{}
			spec1.Name = "test condition"

			spec2 := AlertsPolicyConditionSpec{}
			spec2.Name = "test condition 2"

			spec3 := AlertsPolicyConditionSpec{}
			spec3.Name = "test condition is awesome"

			p.Conditions = []AlertsPolicyCondition{
				{
					Spec: spec1,
				},
				{
					Spec: spec2,
				},
			}
			policyToCompare.Conditions = []AlertsPolicyCondition{
				{
					Spec: spec1,
				},
				{
					Spec: spec3,
				},
			}
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When different number of conditions exist", func() {
		It("should return false", func() {
			spec1 := AlertsPolicyConditionSpec{}
			spec1.Name = "test condition"

			spec2 := AlertsPolicyConditionSpec{}
			spec2.Name = "test condition 2"

			p.Conditions = []AlertsPolicyCondition{
				{
					Spec: spec1,
				},
				{
					Spec: spec2,
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
