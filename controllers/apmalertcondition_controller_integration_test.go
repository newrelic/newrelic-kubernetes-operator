// +build integration

package controllers

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("ApmCondition reconciliation", func() {
	var (
		ctx            context.Context
		r              *ApmAlertConditionReconciler
		condition      *nrv1.ApmAlertCondition
		request        ctrl.Request
		namespacedName types.NamespacedName
		secret         *v1.Secret
		fakeAlertFunc  func(string, string) (interfaces.NewRelicAlertsClient, error)
	)
	BeforeEach(func() {
		ctx = context.Background()

		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}

		alertsClient.CreateConditionStub = func(i int, a alerts.Condition) (*alerts.Condition, error) {
			a.ID = 111
			return &a, nil
		}
		alertsClient.UpdateConditionStub = func(a alerts.Condition) (*alerts.Condition, error) {
			a.ID = 112
			return &a, nil
		}
		alertsClient.ListConditionsStub = func(int) ([]*alerts.Condition, error) {
			var a []*alerts.Condition
			a = append(a, &alerts.Condition{
				ID:   112,
				Name: "Matching APM Condition",
			})
			return a, nil
		}

		alertsClient.DeleteConditionStub = func(int) (*alerts.Condition, error) {
			return &alerts.Condition{}, nil
		}

		fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}

		r = &ApmAlertConditionReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
		}

		condition = &nrv1.ApmAlertCondition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-condition",
				Namespace: "default",
			},
			Spec: nrv1.ApmAlertConditionSpec{
				nrv1.GenericConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "0.9",
							TimeFunction: "all",
						},
					},
					Type:       "apm_app_metric",
					Name:       "APM Condition",
					RunbookURL: "http://test.com/runbook",
					PolicyID:            0,
					ID:                  888,
					Enabled:             true,
					ExistingPolicyID:    42,
					APIKey:              "apikey",
				},
				nrv1.APMSpecificSpec{
					Metric:     "apdex",
					Entities:   []string{"333"},
					UserDefined: alerts.ConditionUserDefined{
						Metric:        "Custom/foo",
						ValueFunction: "average",
					},
					Scope:               "application",
					GCMetric:            "",
					ViolationCloseTimer: 60,
				},
			},
			Status: nrv1.ApmAlertConditionStatus{
				AppliedSpec: &nrv1.ApmAlertConditionSpec{},
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

		Context("and given a new ApmAlertCondition", func() {
			Context("with a valid condition", func() {
				It("should create that condition via the AlertClient", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateConditionCallCount()).To(Equal(0))
				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.ApmAlertCondition
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

					var endStateCondition nrv1.ApmAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a new ApmAlertCondition", func() {
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
					k8sClient.Create(ctx, secret)
				})
				It("should create that condition via the AlertClient", func() {

					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateConditionCallCount()).To(Equal(0))
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

					var endStateCondition nrv1.ApmAlertCondition
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

					var endStateCondition nrv1.ApmAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a ApmAlertCondition that exists in New Relic", func() {
			JustBeforeEach(func() {
				condition = &nrv1.ApmAlertCondition{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-condition",
						Namespace: "default",
					},
					Spec: nrv1.ApmAlertConditionSpec{
						nrv1.GenericConditionSpec{
							Terms: []nrv1.AlertConditionTerm{
								{
									Duration:     "30",
									Operator:     "above",
									Priority:     "critical",
									Threshold:    "0.9",
									TimeFunction: "all",
								},
							},
							Type:       "apm_app_metric",
							Name:       "Matching APM Condition",
							RunbookURL: "http://test.com/runbook",
							PolicyID:            0,
							ID:                  888,
							Enabled:             true,
							ExistingPolicyID:    42,
							APIKey:              "apikey",
						},
						nrv1.APMSpecificSpec{
							Metric:     "apdex",
							Entities:   []string{"333"},
							UserDefined: alerts.ConditionUserDefined{
								Metric:        "Custom/foo",
								ValueFunction: "average",
							},
							Scope:               "application",
							GCMetric:            "",
							ViolationCloseTimer: 60,
						},
					},
					Status: nrv1.ApmAlertConditionStatus{
						AppliedSpec: &nrv1.ApmAlertConditionSpec{},
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

					Expect(alertsClient.CreateConditionCallCount()).To(Equal(0))
				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.ApmAlertCondition
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

					var endStateCondition nrv1.ApmAlertCondition
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

				Expect(alertsClient.CreateConditionCallCount()).To(Equal(1))
				Expect(alertsClient.UpdateConditionCallCount()).To(Equal(0))

				// change the event after creation via reconciliation
				err = k8sClient.Get(ctx, namespacedName, condition)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when condition has been changed", func() {
				BeforeEach(func() {
					condition.Spec.ID = 0
					condition.Spec.Metric = "database"
				})

				It("updates the condition via the client", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					// Only call count for Update is changed from second reconciliation run
					Expect(alertsClient.CreateConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateConditionCallCount()).To(Equal(1))
				})

				It("allows the API to change the ConditionID on the kubernetes object", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.ApmAlertCondition
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

					var endStateCondition nrv1.ApmAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
					Expect(endStateCondition.Status.AppliedSpec.Metric).To(Equal("database"))
				})
			})

			Context("when condition has not changed", func() {
				It("does not make an API call with the client", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateConditionCallCount()).To(Equal(1))
					Expect(alertsClient.UpdateConditionCallCount()).To(Equal(0))
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

		Context("and deleting a ApmAlertCondition", func() {
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

					Expect(alertsClient.CreateConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					Expect(alertsClient.UpdateConditionCallCount()).To(Equal(0))
					Expect(alertsClient.DeleteConditionCallCount()).To(Equal(1))
				})

				It("should return an error when getting the object", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nrv1.ApmAlertCondition
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

					Expect(alertsClient.CreateConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					Expect(alertsClient.UpdateConditionCallCount()).To(Equal(0))
					Expect(alertsClient.DeleteConditionCallCount()).To(Equal(0))

					var endStateCondition nrv1.ApmAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
					Expect(endStateCondition.Name).To(Equal(""))
				})

			})

			Context("when the Alerts API reports no condition found ", func() {

				BeforeEach(func() {
					alertsClient.DeleteConditionStub = func(int) (*alerts.Condition, error) {
						return &alerts.Condition{}, errors.New("resource not found")
					}

				})
				It("should just remove the finalizer and delete", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					Expect(alertsClient.UpdateConditionCallCount()).To(Equal(0))
					Expect(alertsClient.DeleteConditionCallCount()).To(Equal(1))

					var endStateCondition nrv1.ApmAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
					Expect(endStateCondition.Name).To(Equal(""))
				})

			})

		})
	})
})
