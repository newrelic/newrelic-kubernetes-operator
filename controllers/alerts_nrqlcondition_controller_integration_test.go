// +build integration

package controllers

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/testhelpers"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/internal/testutil"
)

var _ = Describe("AlertsNrqlCondition reconciliation", func() {
	var (
		ctx context.Context
		//events         chan string
		r              *AlertsNrqlConditionReconciler
		t              *testing.T
		condition      *nrv1.AlertsNrqlCondition
		request        ctrl.Request
		namespacedName types.NamespacedName
		//expectedEvents []string
		// secret        *v1.Secret
		// fakeAlertFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
		// conditionName string

		k8sClient    client.Client
		alertsClient alerts.Alerts
	)
	BeforeEach(func() {
		ctx = context.Background()
		t = &testing.T{}
		alertsClient = alerts.New(testhelpers.NewIntegrationTestConfig(t))

		// Create a policy to use as the existing policy
		policy := testutil.NewTestAlertsPolicy(t)
		p := alerts.AlertsPolicyInput{}
		p.IncidentPreference = alerts.AlertsIncidentPreference(policy.Spec.IncidentPreference)
		p.Name = policy.Spec.Name

		policyCreateResult, _ := alertsClient.CreatePolicyMutation(policy.Spec.AccountID, p)

		condition = testutil.NewTestAlertsNrqlCondition(t)
		condition.Spec.ExistingPolicyID = policyCreateResult.ID

		// Must come before calling reconciler.Reconcile()
		k8sClient = testutil.AlertsPolicyTestSetup(t)

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      condition.Name,
		}

		request = ctrl.Request{NamespacedName: namespacedName}

		r = &AlertsNrqlConditionReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: interfaces.InitializeAlertsClient,
		}
	})

	Context("when starting with no conditions", func() {

		Context("and given a new AlertsNrqlCondition", func() {
			Context("with a valid condition", func() {
				It("should create the condition", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).To(BeNil())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).To(BeNil())

					conditionNameType := types.NamespacedName{
						Name:      condition.Name,
						Namespace: condition.Namespace,
					}

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
					Expect(err).To(BeNil())

					// Use the alertsClient to query for the result of the reconcile.
					getResult, err := alertsClient.GetNrqlConditionQuery(condition.Spec.AccountID, endStateCondition.Status.ConditionID)
					Expect(err).To(BeNil())

					Expect(endStateCondition.Status.ConditionID).To(Equal(getResult.ID))
				})

				It("has the expected AppliedSpec", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		// 	Context("and given a new AlertsNrqlCondition", func() {
		// 		Context("with a valid condition and a kubernetes secret", func() {
		// 			BeforeEach(func() {
		// 				condition.Spec.APIKey = ""
		// 				condition.Spec.APIKeySecret = nrv1.NewRelicAPIKeySecret{
		// 					Name:      "my-api-key-secret",
		// 					Namespace: "my-namespace",
		// 					KeyName:   "my-api-key",
		// 				}
		//
		// 				secret = &v1.Secret{
		// 					ObjectMeta: metav1.ObjectMeta{
		// 						Name:      "my-api-key-secret",
		// 						Namespace: "my-namespace",
		// 					},
		// 					Data: map[string][]byte{
		// 						"my-api-key": []byte("data_here"),
		// 					},
		// 				}
		// 				k8sClient.Create(ctx, secret)
		// 			})
		// 			It("should create that condition via the AlertClient", func() {
		//
		// 				err := k8sClient.Create(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
		// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
		// 			})
		// 			AfterEach(func() {
		// 				//k8sClient.Delete(ctx, secret)
		//
		// 			})
		//
		// 			It("updates the ConditionID on the kubernetes object", func() {
		// 				err := k8sClient.Create(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(BeNil())
		// 				Expect(endStateCondition.Status.ConditionID).To(Equal(111))
		// 			})
		//
		// 			It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
		// 				err := k8sClient.Create(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(BeNil())
		// 				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
		// 			})
		// 		})
		// 	})
		//
		// 	Context("and given a AlertsNrqlCondition that exists in New Relic", func() {
		// 		JustBeforeEach(func() {
		// 			condition = &nrv1.AlertsNrqlCondition{
		// 				ObjectMeta: metav1.ObjectMeta{
		// 					Name:      conditionName,
		// 					Namespace: "default",
		// 				},
		// 				Spec: nrv1.AlertsNrqlConditionSpec{
		// 					Terms: []nrv1.AlertsNrqlConditionTerm{
		// 						{
		// 							Operator:             alerts.NrqlConditionOperators.Above,
		// 							Priority:             alerts.NrqlConditionPriorities.Critical,
		// 							Threshold:            "5",
		// 							ThresholdDuration:    60,
		// 							ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
		// 						},
		// 					},
		// 					Nrql: alerts.NrqlConditionQuery{
		// 						Query:            "SELECT 1 FROM MyEvents",
		// 						EvaluationOffset: 5,
		// 					},
		// 					Type:               "NRQL",
		// 					Name:               "NRQL Condition matches",
		// 					RunbookURL:         "http://test.com/runbook",
		// 					ID:                 777,
		// 					ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
		// 					ViolationTimeLimit: alerts.NrqlConditionViolationTimeLimits.OneHour,
		// 					ExpectedGroups:     2,
		// 					IgnoreOverlap:      true,
		// 					Enabled:            true,
		// 					ExistingPolicyID:   42,
		// 					APIKey:             "nraa-key",
		// 				},
		// 				Status: nrv1.AlertsNrqlConditionStatus{
		// 					AppliedSpec: &nrv1.AlertsNrqlConditionSpec{},
		// 					ConditionID: 0,
		// 				},
		// 			}
		// 		})
		//
		// 		Context("with a valid condition", func() {
		//
		// 			It("does not create a new condition", func() {
		// 				err := k8sClient.Create(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(0))
		// 			})
		//
		// 			It("updates the ConditionID on the kubernetes object", func() {
		// 				err := k8sClient.Create(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(BeNil())
		// 				Expect(endStateCondition.Status.ConditionID).To(Equal(112))
		// 			})
		//
		// 			It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
		// 				err := k8sClient.Create(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(BeNil())
		// 				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
		// 			})
		//
		// 		})
		// 	})
		//
		// 	Context("and condition has already been created", func() {
		// 		BeforeEach(func() {
		// 			err := k8sClient.Create(ctx, condition)
		// 			Expect(err).ToNot(HaveOccurred())
		//
		// 			// call reconcile
		// 			_, err = r.Reconcile(request)
		// 			Expect(err).ToNot(HaveOccurred())
		//
		// 			Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
		// 			Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
		//
		// 			// change the event after creation via reconciliation
		// 			err = k8sClient.Get(ctx, namespacedName, condition)
		// 			Expect(err).ToNot(HaveOccurred())
		// 		})
		//
		// 		Context("when condition has been changed", func() {
		// 			BeforeEach(func() {
		// 				condition.Spec.ID = 0
		// 				condition.Spec.Nrql.Query = "SELECT 1 FROM NewEventType"
		// 			})
		//
		// 			It("updates the condition via the client", func() {
		// 				err := k8sClient.Update(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// Only call count for Update is changed from second reconciliation run
		// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
		// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(1))
		// 			})
		//
		// 			It("allows the API to change the ConditionID on the kubernetes object", func() {
		// 				err := k8sClient.Update(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(BeNil())
		// 				Expect(endStateCondition.Status.ConditionID).To(Equal(112))
		// 			})
		//
		// 			It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
		// 				err := k8sClient.Update(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(BeNil())
		// 				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
		// 			})
		// 		})
		//
		// 		Context("when condition has not changed", func() {
		// 			It("does not make an API call with the client", func() {
		// 				err := k8sClient.Update(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
		// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
		// 			})
		// 		})
		// 	})
		//
		// 	AfterEach(func() {
		// 		// Delete the condition
		// 		err := k8sClient.Delete(ctx, condition)
		// 		Expect(err).ToNot(HaveOccurred())
		//
		// 		// Need to call reconcile to delete finalizer
		// 		_, err = r.Reconcile(request)
		// 		Expect(err).ToNot(HaveOccurred())
		// 	})
		// })
		//
		// Context("When starting with an existing condition", func() {
		//
		// 	Context("and deleting a AlertsNrqlCondition", func() {
		// 		BeforeEach(func() {
		// 			err := k8sClient.Create(ctx, condition)
		// 			Expect(err).ToNot(HaveOccurred())
		//
		// 			// call reconcile
		// 			_, err = r.Reconcile(request)
		// 			Expect(err).ToNot(HaveOccurred())
		//
		// 			// change the event after creation via reconciliation
		// 			err = k8sClient.Get(ctx, namespacedName, condition)
		// 			Expect(err).ToNot(HaveOccurred())
		//
		// 		})
		// 		Context("with a valid condition", func() {
		// 			It("should delete that condition via the AlertClient", func() {
		// 				err := k8sClient.Delete(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
		// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
		// 				Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))
		// 			})
		//
		// 			It("should return an error when getting the object", func() {
		// 				err := k8sClient.Delete(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(HaveOccurred())
		// 			})
		//
		// 		})
		// 		Context("with a condition with no condition ID", func() {
		//
		// 			BeforeEach(func() {
		// 				condition.Status.ConditionID = 0
		// 				err := k8sClient.Update(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 			})
		// 			It("should just remove the finalizer and delete", func() {
		// 				err := k8sClient.Delete(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
		// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
		// 				Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(0))
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(HaveOccurred())
		// 				Expect(endStateCondition.Name).To(Equal(""))
		// 			})
		//
		// 		})
		//
		// 		Context("when the Alerts API reports no condition found ", func() {
		//
		// 			BeforeEach(func() {
		// 				alertsClient.DeleteNrqlConditionStub = func(int) (*alerts.NrqlCondition, error) {
		// 					return &alerts.NrqlCondition{}, errors.New("resource not found")
		// 				}
		//
		// 			})
		// 			It("should just remove the finalizer and delete", func() {
		// 				err := k8sClient.Delete(ctx, condition)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				// call reconcile
		// 				_, err = r.Reconcile(request)
		// 				Expect(err).ToNot(HaveOccurred())
		//
		// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
		// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
		// 				Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))
		//
		// 				var endStateCondition nrv1.AlertsNrqlCondition
		// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
		// 				Expect(err).To(HaveOccurred())
		// 				Expect(endStateCondition.Name).To(Equal(""))
		// 			})
		//
		// 		})
		// })
	})

	//Context("When reading the API key from a secret", func() {
	//	It("should read the secret", func() {
	//		secret := &v1.Secret{
	//			ObjectMeta: metav1.ObjectMeta{
	//				Name:      "newrelic",
	//				Namespace: "default",
	//			},
	//			Data: map[string][]byte{
	//				"api_key": []byte("api-key"),
	//			},
	//		}
	//		err := k8sClient.Create(ctx, secret)
	//
	//		Expect(err).ToNot(HaveOccurred())
	//		_, err = r.Reconcile(request)
	//
	//
	//		Expect("this").To(Equal("that"))
	//
	//
	//	})
	//})

})
