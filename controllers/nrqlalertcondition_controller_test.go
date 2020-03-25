package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	nralertsv1beta1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1beta1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("NrqlCondition reconciliation", func() {
	var (
		ctx context.Context
		//events         chan string
		r              *NrqlAlertConditionReconciler
		condition      *nralertsv1beta1.NrqlAlertCondition
		request        ctrl.Request
		namespacedName types.NamespacedName
		//expectedEvents []string
	)

	BeforeEach(func() {
		ctx = context.Background()

		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}

		r = &NrqlAlertConditionReconciler{
			Client: k8sClient,
			Log:    logf.Log,
			Alerts: alertsClient,
		}

		condition = &nralertsv1beta1.NrqlAlertCondition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-condition",
				Namespace: "default",
			},
			Spec: nralertsv1beta1.NrqlAlertConditionSpec{
				Terms: []nralertsv1beta1.AlertConditionTerm{
					nralertsv1beta1.AlertConditionTerm{
						Duration:     resource.MustParse("30"),
						Operator:     "Above",
						Priority:     "Critical",
						Threshold:    resource.MustParse("5"),
						TimeFunction: "All",
					},
				},
				Nrql: nralertsv1beta1.NrqlQuery{
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
				ExistingPolicyId:    42,
			},
			Status: nralertsv1beta1.NrqlAlertConditionStatus{
				AppliedSpec: &nralertsv1beta1.NrqlAlertConditionSpec{},
				ConditionID: 0,
			},
		}
		condition = &nralertsv1beta1.NrqlAlertCondition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-condition",
				Namespace: "default",
			},
			Spec: nralertsv1beta1.NrqlAlertConditionSpec{
				Terms: []nralertsv1beta1.AlertConditionTerm{
					nralertsv1beta1.AlertConditionTerm{
						Duration:     resource.MustParse("30"),
						Operator:     "Above",
						Priority:     "Critical",
						Threshold:    resource.MustParse("5"),
						TimeFunction: "All",
					},
				},
				Nrql: nralertsv1beta1.NrqlQuery{
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
				ExistingPolicyId:    42,
			},
			Status: nralertsv1beta1.NrqlAlertConditionStatus{
				AppliedSpec: &nralertsv1beta1.NrqlAlertConditionSpec{},
				ConditionID: 0,
			},
		}
		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "test-condition",
		}
		request = ctrl.Request{NamespacedName: namespacedName}

		alertsClient.CreateNrqlConditionStub = func(a alerts.NrqlCondition) (*alerts.NrqlCondition, error) {
			a.ID = 111
			return &a, nil
		}
		alertsClient.UpdateNrqlConditionStub = func(a alerts.NrqlCondition) (*alerts.NrqlCondition, error) {
			a.ID = 112
			return &a, nil
		}
		alertsClient.ListNrqlConditionsStub = func(int) (*[]alerts.NrqlCondition, error) {
			var a []alerts.NrqlCondition
			a = append(a, alerts.NrqlCondition{
				ID: 112,
				Name: "NRQL Condition matches",
			})
			return &a, nil
		}
	})

	Context("when given a new NrqlAlertCondition", func() {
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

				var endStateCondition nralertsv1beta1.NrqlAlertCondition
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

				var endStateCondition nralertsv1beta1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
			})
		})
	})

	Context("when given a NrqlAlertCondition that exists in New Relic", func() {
		JustBeforeEach(func() {
			condition = &nralertsv1beta1.NrqlAlertCondition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-condition",
					Namespace: "default",
				},
				Spec: nralertsv1beta1.NrqlAlertConditionSpec{
					Terms: []nralertsv1beta1.AlertConditionTerm{
						nralertsv1beta1.AlertConditionTerm{
							Duration:     resource.MustParse("30"),
							Operator:     "Above",
							Priority:     "Critical",
							Threshold:    resource.MustParse("5"),
							TimeFunction: "All",
						},
					},
					Nrql: nralertsv1beta1.NrqlQuery{
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
					ExistingPolicyId:    42,
				},
				Status: nralertsv1beta1.NrqlAlertConditionStatus{
					AppliedSpec: &nralertsv1beta1.NrqlAlertConditionSpec{},
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

				var endStateCondition nralertsv1beta1.NrqlAlertCondition
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

				var endStateCondition nralertsv1beta1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
			})

		})
	})

	Context("when condition has already been created", func() {
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

				var endStateCondition nralertsv1beta1.NrqlAlertCondition
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

				var endStateCondition nralertsv1beta1.NrqlAlertCondition
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
