// +build integration

package controllers

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
	"github.com/newrelic/newrelic-kubernetes-operator/internal/testutil"
)

var _ = Describe("AlertsNrqlCondition reconciliation", func() {
	var (
		ctx                  context.Context
		r                    *AlertsNrqlConditionReconciler
		t                    *testing.T
		condition            *nrv1.AlertsNrqlCondition
		request              ctrl.Request
		namespacedName       types.NamespacedName
		secret               *v1.Secret
		mockAlertsClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
		conditionName        string

		k8sClient        client.Client
		mockAlertsClient *interfacesfakes.FakeNewRelicAlertsClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		t = &testing.T{}
		k8sClient = testutil.AlertsPolicyTestSetup(t)
		mockAlertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}

		mockAlertsClient.CreateNrqlConditionStaticMutationStub = func(accountID int, policyID string, a alerts.NrqlConditionInput) (*alerts.NrqlAlertCondition, error) {
			condition := &alerts.NrqlAlertCondition{
				ID: "111",
			}
			return condition, nil
		}

		mockAlertsClient.UpdateNrqlConditionStaticMutationStub = func(accountID int, conditionID string, a alerts.NrqlConditionInput) (*alerts.NrqlAlertCondition, error) {
			condition := &alerts.NrqlAlertCondition{
				ID: "112",
			}
			return condition, nil
		}

		mockAlertsClient.CreateNrqlConditionBaselineMutationStub = func(accountID int, policyID string, a alerts.NrqlConditionInput) (*alerts.NrqlAlertCondition, error) {
			condition := &alerts.NrqlAlertCondition{
				ID: "111",
			}
			return condition, nil
		}

		mockAlertsClient.UpdateNrqlConditionBaselineMutationStub = func(accountID int, conditionID string, a alerts.NrqlConditionInput) (*alerts.NrqlAlertCondition, error) {
			condition := &alerts.NrqlAlertCondition{
				ID: "112",
			}
			return condition, nil
		}

		mockAlertsClient.SearchNrqlConditionsQueryStub = func(accountID int, searchCriteria alerts.NrqlConditionsSearchCriteria) ([]*alerts.NrqlAlertCondition, error) {
			var a []*alerts.NrqlAlertCondition
			condition := &alerts.NrqlAlertCondition{
				ID: "112",
			}
			condition.Name = "NRQL Condition matches"
			a = append(a, condition)
			return a, nil
		}

		mockAlertsClient.DeleteConditionMutationStub = func(accountID int, conditionID string) (string, error) {
			return "", nil
		}

		mockAlertsClientFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return mockAlertsClient, nil
		}

		condition = testutil.NewTestAlertsNrqlCondition(t)
		conditionName = condition.Name

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      conditionName,
		}

		request = ctrl.Request{NamespacedName: namespacedName}

		err := ignoreAlreadyExists(k8sClient.Create(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		}))
		Expect(err).ToNot(HaveOccurred())

		newrelicAgent := newrelic.Application{}

		r = &AlertsNrqlConditionReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: mockAlertsClientFunc,
			NewRelicAgent:   newrelicAgent,
		}
	})

	Context("when starting with no conditions", func() {
		Context("and given a new AlertsNrqlCondition", func() {
			Context("with a valid condition", func() {
				It("should create the condition", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).To(BeNil())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(0))
				})

				It("has the expected AppliedSpec", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a new AlertsNrqlCondition", func() {
			Context("with a valid condition and a kubernetes secret", func() {
				BeforeEach(func() {
					condition.Spec.APIKey = ""
					condition.Spec.APIKeySecret = nrv1.NewRelicAPIKeySecret{
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

					err := k8sClient.Create(ctx, secret)
					Expect(err).ToNot(HaveOccurred())
				})

				It("should create that condition via the AlertClient", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(0))
				})

				AfterEach(func() {
					//k8sClient.Delete(ctx, secret)
				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.ConditionID).To(Equal("111"))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a AlertsNrqlCondition that exists in New Relic", func() {
			JustBeforeEach(func() {
				spec := nrv1.AlertsNrqlConditionSpec{}
				spec.ValueFunction = &alerts.NrqlConditionValueFunctions.SingleValue
				spec.ViolationTimeLimit = alerts.NrqlConditionViolationTimeLimits.OneHour
				spec.ExpectedGroups = 2
				spec.IgnoreOverlap = true
				spec.Enabled = true
				spec.ExistingPolicyID = "42"
				spec.APIKey = "nraa-key"
				spec.Terms = []nrv1.AlertsNrqlConditionTerm{
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
				spec.Name = "NRQL Condition matches"
				spec.RunbookURL = "http://test.com/runbook"
				spec.ID = 777

				condition = &nrv1.AlertsNrqlCondition{
					ObjectMeta: metav1.ObjectMeta{
						Name:      conditionName,
						Namespace: "default",
					},
					Spec: spec,
					Status: nrv1.AlertsNrqlConditionStatus{
						AppliedSpec: &nrv1.AlertsNrqlConditionSpec{},
						ConditionID: "0",
					},
				}
			})

			Context("with a valid condition", func() {
				It("does not create a new condition", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(0))
				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.ConditionID).To(Equal("112"))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and condition has already been created", func() {
			BeforeEach(func() {
				err := k8sClient.Create(ctx, condition)
				Expect(err).ToNot(HaveOccurred())

				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
				Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(0))

				err = k8sClient.Get(ctx, namespacedName, condition)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when condition has been changed", func() {
				BeforeEach(func() {
					condition.Spec.ID = 0
					condition.Spec.Nrql.Query = "SELECT 1 FROM NewEventType"
				})

				It("updates the condition via the client", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(1))
				})

				It("allows the API to change the ConditionID on the kubernetes object", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.ConditionID).To(Equal("112"))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})

			Context("when condition has not changed", func() {
				It("does not make an API call with the client", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(0))
				})
			})
		})

		Context("and given a new baseline condition", func() {
			Context("with a valid condition", func() {
				BeforeEach(func() {
					condition.Spec.BaselineDirection = &alerts.NrqlBaselineDirections.UpperAndLower
				})
				It("should create the condition", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).To(BeNil())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionBaselineMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionBaselineMutationCallCount()).To(Equal(0))
				})

				It("has the expected AppliedSpec", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		AfterEach(func() {
			// Delete the condition
			err := k8sClient.Delete(ctx, condition)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When starting with an existing condition", func() {
		Context("and deleting a AlertsNrqlCondition", func() {
			BeforeEach(func() {
				err := k8sClient.Create(ctx, condition)
				Expect(err).ToNot(HaveOccurred())

				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				err = k8sClient.Get(ctx, namespacedName, condition)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("with a valid condition", func() {
				It("should delete that condition via the AlertClient", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(0))
					Expect(mockAlertsClient.DeleteConditionMutationCallCount()).To(Equal(1))
				})

				It("should return an error when getting the object", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("with a condition with no condition ID", func() {
				BeforeEach(func() {
					condition.Status.ConditionID = "0"
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())
				})

				It("should just remove the finalizer and delete", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(0))
					Expect(mockAlertsClient.DeleteNrqlConditionCallCount()).To(Equal(0))

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
					Expect(endStateCondition.Name).To(Equal(""))
				})
			})

			Context("when the Alerts API reports no condition found ", func() {
				BeforeEach(func() {
					mockAlertsClient.DeleteNrqlConditionStub = func(int) (*alerts.NrqlCondition, error) {
						return &alerts.NrqlCondition{}, errors.New("resource not found")
					}
				})

				It("should just remove the finalizer and delete", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(mockAlertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(1))
					Expect(mockAlertsClient.UpdateNrqlConditionStaticMutationCallCount()).To(Equal(0))
					Expect(mockAlertsClient.DeleteConditionMutationCallCount()).To(Equal(1))

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
					Expect(endStateCondition.Name).To(Equal(""))
				})
			})
		})
	})

	Context("When reading the API key from a secret", func() {
		It("should read the secret", func() {
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "newrelic",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api_key": []byte("api-key"),
				},
			}
			err := k8sClient.Create(ctx, secret)

			Expect(err).ToNot(HaveOccurred())
			_, err = r.Reconcile(request)
		})
	})
})
