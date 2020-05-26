// +build integration

package v1

import (
	"errors"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("ValidateCreate", func() {
	var (
		r            AlertsNrqlCondition
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

		r = AlertsNrqlCondition{
			Spec: AlertsNrqlConditionSpec{
				Terms: []AlertsNrqlConditionTerm{
					{
						Duration:     resource.MustParse("30"),
						Operator:     "above",
						Priority:     "critical",
						Threshold:    resource.MustParse("5"),
						TimeFunction: "all",
					},
				},
				Nrql: AlertsNrqlConditionQuery{
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
				APIKey:              "api-key",
				Region:              "us",
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

	Context("when given an API key in a secret that can't be read", func() {
		It("should return an error", func() {
			r.Spec.APIKey = ""
			r.Spec.APIKeySecret = NewRelicAPIKeySecret{
				Name:      "my-api-key-secret",
				Namespace: "my-namespace",
				KeyName:   "my-api-key",
			}
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when given a NRQL condition without required field region", func() {
		It("should reject resource creation", func() {
			r.Spec.Region = ""
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when given a NRQL condition without required field ExistingPolicyId", func() {
		It("should reject resource creation", func() {
			r.Spec.ExistingPolicyID = 0
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		Context("when missing multiple required fields, include all messages in one error", func() {
			It("should reject resource creation", func() {
				r.Spec.Region = ""
				r.Spec.ExistingPolicyID = 0
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errors.New("region and existing_policy_id must be set")))
			})
		})
	})

	Describe("CheckExistingPolicyID", func() {
		BeforeEach(func() {})

		Context("With a valid API Key", func() {
			BeforeEach(func() {})

			It("verifies existing policies exist", func() {
				err := r.CheckExistingPolicyID()
				Expect(err).To(BeNil())
				Expect(r.Spec.ExistingPolicyID).To(Equal(42))
			})
		})

		Context("With an invalid API Key", func() {
			BeforeEach(func() {
				alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
					return nil, errors.New("401 response returned: The API key provided is invalid")
				}
			})

			It("returns an error", func() {
				err := r.CheckExistingPolicyID()
				Expect(err).To(Not(BeNil()))
			})
		})
	})

	Describe("InvalidPolicyID", func() {
		Context("With a Policy ID that does not exist", func() {
			BeforeEach(func() {
				r.Spec.ExistingPolicyID = 0
			})

			It("returns an error", func() {
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
