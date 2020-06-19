// +build integration

package v1

import (
	"context"
	"errors"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("ValidateCreate", func() {
	var (
		r            NrqlAlertCondition
		alertsClient *interfacesfakes.FakeNewRelicAlertsClient
		secret       *v1.Secret
		ctx          context.Context
	)

	BeforeEach(func() {
		err := ignoreAlreadyExists(testk8sClient.Create(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		}))
		Expect(err).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		k8Client = testk8sClient
		ctx = context.Background()
		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}
		fakeAlertFunc := func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}
		alertClientFunc = fakeAlertFunc

		r = NrqlAlertCondition{
			Spec: NrqlAlertConditionSpec{
				GenericConditionSpec{
					Terms: []AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "5",
							TimeFunction: "all",
						},
					},
					Type:             "NRQL",
					Name:             "NRQL Condition",
					RunbookURL:       "http://test.com/runbook",
					ID:               777,
					Enabled:          true,
					ExistingPolicyID: 42,
					APIKey:           "api-key",
					Region:           "us",
				},
				NrqlSpecificSpec{
					ViolationCloseTimer: 60,
					ExpectedGroups:      2,
					IgnoreOverlap:       true,
					ValueFunction:       "max",
					Nrql: NrqlQuery{
						Query:      "SELECT 1 FROM MyEvents",
						SinceValue: "5",
					},
				},
			},
		}

		// TODO: Make this a true integration test if possible
		alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
			return &alerts.Policy{
				ID: 42,
			}, nil
		}
	})

	Context("ValidateCreate", func() {
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
				Expect(ignoreAlreadyExists(k8Client.Create(ctx, secret))).To(Succeed())
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

	Context("ValidateUpdate", func() {
		Context("When deleting an existing nrql Condition with a delete policy", func() {
			var update NrqlAlertCondition

			BeforeEach(func() {
				currentTime := metav1.Time{Time: time.Now()}
				//make copy of existing object to update
				r.DeepCopyInto(&update)

				update.SetDeletionTimestamp(&currentTime)
				alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
					return &alerts.Policy{}, errors.New("no alert policy found for id 49092")
				}
			})

			It("Should allow the deletion anyway", func() {
				err := update.ValidateUpdate(&r)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

})
