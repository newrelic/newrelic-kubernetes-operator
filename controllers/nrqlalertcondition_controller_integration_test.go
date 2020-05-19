// +build integration

package controllers

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	operatorAPI "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("NrqlCondition reconciliation", func() {
	var (
		ctx            context.Context
		r              *NrqlConditionReconciler
		condition      *operatorAPI.NrqlConditionSchema
		request        ctrl.Request
		namespacedName types.NamespacedName
		secret         *v1.Secret
		fakeAlertFunc  func(string, string) (interfaces.NewRelicAlertsClient, error)
	)
	BeforeEach(func() {
		ctx = context.Background()

		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}

		alertsClient.CreateNrqlConditionStub = func(i int, a alerts.NrqlCondition) (*alerts.NrqlCondition, error) {
			a.ID = 111
			return &a, nil
		}
		alertsClient.UpdateNrqlConditionStub = func(a alerts.NrqlCondition) (*alerts.NrqlCondition, error) {
			a.ID = 112
			return &a, nil
		}
		alertsClient.ListNrqlConditionsStub = func(int) ([]*alerts.NrqlCondition, error) {
			var a []*alerts.NrqlCondition
			a = append(a, &alerts.NrqlCondition{
				ID:   112,
				Name: "NRQL Condition matches",
			})
			return a, nil
		}

		alertsClient.DeleteNrqlConditionStub = func(int) (*alerts.NrqlCondition, error) {
			return &alerts.NrqlCondition{}, nil
		}

		fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}

		r = &NrqlConditionReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
		}

		condition = &operatorAPI.NrqlConditionSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-condition",
				Namespace: "default",
			},
			Spec: operatorAPI.NrqlConditionSpec{
				Terms: []operatorAPI.ConditionTerm{
					{
						Duration:     resource.MustParse("30"),
						Operator:     "Above",
						Priority:     "Critical",
						Threshold:    resource.MustParse("5"),
						TimeFunction: "All",
					},
				},
				Nrql: operatorAPI.NrqlQuery{
					Query:      "SELECT 1 FROM MyEvents",
					SinceValue: "5",
				},
				Type:                "NRQL",
				Name:                "NRQL Condition",
				RunbookURL:          "http://test.com/runbook",
				ValueFunction:       "Over",
				ID:                  777,
				ViolationCloseTimer: 60,
				ExpectedGroups:      2,
				IgnoreOverlap:       true,
				Enabled:             true,
				ExistingPolicyID:    42,
				APIKey:              "nraa-key",
			},
			Status: operatorAPI.NrqlConditionStatus{
				AppliedSpec: &operatorAPI.NrqlConditionSpec{},
				ConditionID: 0,
			},
		}
		condition = &operatorAPI.NrqlConditionSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-condition",
				Namespace: "default",
			},
			Spec: operatorAPI.NrqlConditionSpec{
				Terms: []operatorAPI.ConditionTerm{
					{
						Duration:     resource.MustParse("30"),
						Operator:     "Above",
						Priority:     "Critical",
						Threshold:    resource.MustParse("5"),
						TimeFunction: "All",
					},
				},
				Nrql: operatorAPI.NrqlQuery{
					Query:      "SELECT 1 FROM MyEvents",
					SinceValue: "5",
				},
				Type:                "NRQL",
				Name:                "NRQL Condition",
				RunbookURL:          "http://test.com/runbook",
				ValueFunction:       "Over",
				ID:                  777,
				ViolationCloseTimer: 60,
				ExpectedGroups:      2,
				IgnoreOverlap:       true,
				Enabled:             true,
				ExistingPolicyID:    42,
				APIKey:              "nraa-key",
			},
			Status: operatorAPI.NrqlConditionStatus{
				AppliedSpec: &operatorAPI.NrqlConditionSpec{},
				ConditionID: 0,
			},
		}
		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "test-condition",
		}
		request = ctrl.Request{NamespacedName: namespacedName}

	})

	Context("When starting with no conditions", func() {

		Context("and given a new NrqlCondition", func() {
			Context("with a valid condition", func() {
				It("should create that condition via the AlertClient", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.ConditionID).To(Equal(111))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a new NrqlCondition", func() {
			Context("with a valid condition and a kubernetes secret", func() {
				BeforeEach(func() {
					condition.Spec.APIKey = ""
					condition.Spec.APIKeySecret = operatorAPI.NewRelicAPIKeySecret{
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
					k8sClient.Create(ctx, secret)
				})
				It("should create that condition via the AlertClient", func() {

					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
				})
				AfterEach(func() {
					//k8sClient.Delete(ctx, secret)

				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.ConditionID).To(Equal(111))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a NrqlCondition that exists in New Relic", func() {
			JustBeforeEach(func() {
				condition = &operatorAPI.NrqlConditionSchema{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-condition",
						Namespace: "default",
					},
					Spec: operatorAPI.NrqlConditionSpec{
						Terms: []operatorAPI.ConditionTerm{
							{
								Duration:     resource.MustParse("30"),
								Operator:     "Above",
								Priority:     "Critical",
								Threshold:    resource.MustParse("5"),
								TimeFunction: "All",
							},
						},
						Nrql: operatorAPI.NrqlQuery{
							Query:      "SELECT 1 FROM MyEvents",
							SinceValue: "5",
						},
						Type:                "NRQL",
						Name:                "NRQL Condition matches",
						RunbookURL:          "http://test.com/runbook",
						ValueFunction:       "Over",
						ID:                  777,
						ViolationCloseTimer: 60,
						ExpectedGroups:      2,
						IgnoreOverlap:       true,
						Enabled:             true,
						ExistingPolicyID:    42,
						APIKey:              "nraa-key",
					},
					Status: operatorAPI.NrqlConditionStatus{
						AppliedSpec: &operatorAPI.NrqlConditionSpec{},
						ConditionID: 0,
					},
				}
			})

			Context("with a valid condition", func() {

				It("does not create a new condition", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(0))
				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.ConditionID).To(Equal(112))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
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

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))

				// change the event after creation via reconciliation
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

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					// Only call count for Update is changed from second reconciliation run
					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(1))
				})

				It("allows the API to change the ConditionID on the kubernetes object", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.ConditionID).To(Equal(112))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})

			Context("when condition has not changed", func() {
				It("does not make an API call with the client", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
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

		Context("and deleting a NrqlCondition", func() {
			BeforeEach(func() {
				err := k8sClient.Create(ctx, condition)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				// change the event after creation via reconciliation
				err = k8sClient.Get(ctx, namespacedName, condition)
				Expect(err).ToNot(HaveOccurred())

			})
			Context("with a valid condition", func() {
				It("should delete that condition via the AlertClient", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
					Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))
				})

				It("should return an error when getting the object", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
				})

			})
			Context("with a condition with no condition ID", func() {

				BeforeEach(func() {
					condition.Status.ConditionID = 0
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

				})
				It("should just remove the finalizer and delete", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
					Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(0))

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
					Expect(endStateCondition.Name).To(Equal(""))
				})

			})

			Context("when the Alerts API reports no condition found ", func() {

				BeforeEach(func() {
					alertsClient.DeleteNrqlConditionStub = func(int) (*alerts.NrqlCondition, error) {
						return &alerts.NrqlCondition{}, errors.New("resource not found")
					}

				})
				It("should just remove the finalizer and delete", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
					Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))

					var endStateCondition operatorAPI.NrqlConditionSchema
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
					Expect(endStateCondition.Name).To(Equal(""))
				})

			})

		})
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
