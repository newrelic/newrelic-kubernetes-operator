// +build integration

package v1

import (


	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Equals", func() {
	var (
		p PolicySpec
		policyToCompare PolicySpec
		output bool
		condition PolicyCondition
	)
	BeforeEach(func() {
		condition = PolicyCondition{
			Name:      "policy-name",
			Namespace: "default",
			Spec:      NrqlAlertConditionSpec{
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
			},
		}
	 p = PolicySpec{
		 IncidentPreference: "PER_POLICY",
		 Name:               "test-policy",
		 APIKey:             "112233",
		 APIKeySecret:       NewRelicAPIKeySecret{
			 Name:      "secret",
			 Namespace: "default",
			 KeyName:   "api-key",
		 },
		 Region:             "us",
		 Conditions:         []PolicyCondition{condition},
	 }
	 policyToCompare = PolicySpec{
		 IncidentPreference: "PER_POLICY",
		 Name:               "test-policy",
		 APIKey:             "112233",
		 APIKeySecret:       NewRelicAPIKeySecret{
			 Name:      "secret",
			 Namespace: "default",
			 KeyName:   "api-key",
		 },
		 Region:             "us",
		 Conditions:         []PolicyCondition{condition},
	 }
	})


	Context("When equal", func() {
		It("should return true", func() {
			output = p.Equals(policyToCompare)
			Expect(output).To (BeTrue())
		})
	})

	Context("When condition hash matches", func() {

		It("should return true", func() {
				output = p.Equals(policyToCompare)
			Expect(output).To(BeTrue())
		})
	})
	Context("When condition hash matches but condition name doesn't", func() {

		It("should return true", func() {
			p.Conditions = []PolicyCondition{
				{
					Name:      "",
					Namespace: "",
					Spec:      NrqlAlertConditionSpec{
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
					Spec:      NrqlAlertConditionSpec{
						Name: "test condition 222",
					},
				},
			}
			output = p.Equals(policyToCompare)
			Expect(output).ToNot(BeTrue())
		})
	})

	Context("When one condition hash doesn't match matches but the other does", func() {

		It("should return false", func() {
			p.Conditions = []PolicyCondition{
				{
					Spec:      NrqlAlertConditionSpec{
						Name: "test condition",
					},
				},
				{
					Spec:      NrqlAlertConditionSpec{
						Name: "test condition 2",
					},
				},
			}
			policyToCompare.Conditions = []PolicyCondition{
				{
					Spec:      NrqlAlertConditionSpec{
						Name: "test condition",
					},
				},
				{
					Spec:      NrqlAlertConditionSpec{
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
					Spec:      NrqlAlertConditionSpec{
						Name: "test condition",
					},
				},
				{
					Spec:      NrqlAlertConditionSpec{
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

