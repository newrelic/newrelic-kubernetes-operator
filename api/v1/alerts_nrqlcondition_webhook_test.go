// +build integration

package v1

import (
	"context"
	"errors"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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

		spec := AlertsNrqlConditionSpec{}
		spec.Terms = []AlertsNrqlConditionTerm{
			{
				Operator:             alerts.AlertsNRQLConditionTermsOperatorTypes.ABOVE,
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
		spec.ViolationTimeLimit = alerts.NrqlConditionViolationTimeLimits.OneHour
		spec.ID = 777
		spec.ExpectedGroups = 2
		spec.IgnoreOverlap = true
		spec.Enabled = true
		spec.ExistingPolicyID = "42"
		spec.APIKey = "api-key"
		spec.Region = "us"

		r = AlertsNrqlCondition{
			Spec: spec,
		}

		err := ignoreAlreadyExists(testk8sClient.Create(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		}))
		Expect(err).ToNot(HaveOccurred())

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
			err := k8Client.Create(context.Background(), secret)
			Expect(err).ToNot(HaveOccurred())

			err = r.ValidateCreate()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			k8Client.Delete(context.Background(), secret)
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
			r.Spec.ExistingPolicyID = ""
			err := r.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})

		Context("when missing multiple required fields, include all messages in one error", func() {
			It("should reject resource creation", func() {
				r.Spec.Region = ""
				r.Spec.ExistingPolicyID = ""
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(errors.New("region and existing_policy_id must be set")))
			})
		})
	})

	Context("when updating an existing NRQL condition", func() {
		Context("and changing the type from static to baseline", func() {
			It("should fail validation", func() {
				updated := r.DeepCopy()
				updated.Spec.BaselineDirection = &alerts.NrqlBaselineDirections.UpperAndLower
				err := updated.ValidateUpdate(&r)
				Expect(err.Error()).To(Equal("cannot change between condition types, you must delete and create a new alert"))
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
				Expect(r.Spec.ExistingPolicyID).To(Equal("42"))
			})
		})

		Context("With an invalid API Key", func() {
			BeforeEach(func() {
				alertsClient.QueryPolicyStub = func(int, string) (*alerts.AlertsPolicy, error) {
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
				r.Spec.ExistingPolicyID = ""
			})

			It("returns an error", func() {
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
