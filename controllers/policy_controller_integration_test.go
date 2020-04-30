// +build integration

package controllers

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("policy reconciliation", func() {
	var (
		ctx            context.Context
		r              *PolicyReconciler
		policy         *nrv1.Policy
		conditionSpec  *nrv1.NrqlAlertConditionSpec
		request        ctrl.Request
		namespacedName types.NamespacedName
		conditionName  types.NamespacedName
		//expectedEvents []string
		//secret        *v1.Secret
		fakeAlertFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	)

	BeforeEach(func() {
		ctx = context.Background()

		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}

		fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}

		alertsClient.CreatePolicyStub = func(a alerts.Policy) (*alerts.Policy, error) {
			a.ID = 333
			return &a, nil
		}

		alertsClient.UpdatePolicyStub = func(alerts.Policy) (*alerts.Policy, error) {
			return &alerts.Policy{
				ID: 222,
			}, nil
		}

		r = &PolicyReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
		}

		conditionSpec = &nrv1.NrqlAlertConditionSpec{
			Terms: []nrv1.AlertConditionTerm{
				{
					Duration:     resource.MustParse("30"),
					Operator:     "above",
					Priority:     "critical",
					Threshold:    resource.MustParse("5"),
					TimeFunction: "all",
				},
			},
			Nrql: nrv1.NrqlQuery{
				Query:      "SELECT 1 FROM MyEvents",
				SinceValue: "5",
			},
			Type:                "NRQL",
			Name:                "NRQL Condition",
			RunbookURL:          "http://test.com/runbook",
			ValueFunction:       "max",
			ViolationCloseTimer: 60,
			ExpectedGroups:      2,
			IgnoreOverlap:       true,
			Enabled:             true,
		}

		policy = &nrv1.Policy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-policy",
				Namespace: "default",
			},
			Spec: nrv1.PolicySpec{
				Name:               "test policy",
				APIKey:             "112233",
				IncidentPreference: "PER_POLICY",
				Conditions: []nrv1.NrqlAlertCondition{
					{
						Spec: *conditionSpec,
						Status: nrv1.NrqlAlertConditionStatus{
							AppliedSpec: &nrv1.NrqlAlertConditionSpec{},
						},
					},
				},
			},
			Status: nrv1.PolicyStatus{
				AppliedSpec: &nrv1.PolicySpec{},
				PolicyID:    0,
			},
		}

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "test-policy",
		}
		conditionName = types.NamespacedName{
			Namespace: "default",
			Name:      policy.Name + fmt.Sprintf("%d", nrv1.ComputeHash(conditionSpec)),
		}
		request = ctrl.Request{NamespacedName: namespacedName}

	})

	Context("When starting with no policies", func() {
		Context("when creating a valid policy", func() {
			It("should create that policy", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

			})

			It("updates the policyId on the Policy resource", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				Expect(endStatePolicy.Status.PolicyID).To(Equal(333))

			})
			It("creates the condition with attributes from the Policy yaml", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, conditionName, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Nrql.Query).To(Equal("SELECT 1 FROM MyEvents"))
				Expect(endStateCondition.Spec.Terms[0].Priority).To(Equal("critical"))
				Expect(endStateCondition.Spec.Enabled).To(BeTrue())

			})

			It("creates the condition with inherited attributes from the Policy resource", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, conditionName, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.ExistingPolicyID).To(Equal(333))
				Expect(endStateCondition.Spec.Region).To(Equal(policy.Spec.Region))
				Expect(endStateCondition.Spec.APIKey).To(Equal(policy.Spec.APIKey))

			})
		})
		Context("when the New Relic API returns an error", func() {
			BeforeEach(func() {
				alertsClient.CreatePolicyStub = func(alerts.Policy) (*alerts.Policy, error) {
					return &alerts.Policy{}, errors.New("Any Error Goes Here")
				}
			})

			It("should not update the PolicyID", func() {

				createErr := k8sClient.Create(ctx, policy)
				Expect(createErr).ToNot(HaveOccurred())

				// call reconcile
				_, reconcileErr := r.Reconcile(request)
				Expect(reconcileErr).To(HaveOccurred())

				var endStatePolicy nrv1.Policy
				getErr := k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(getErr).ToNot(HaveOccurred())
				Expect(endStatePolicy.Status.PolicyID).To(Equal(0))

			})
		})

		Context("when creating a valid policy with conditions", func() {
			It("should create the conditions", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				//call reconcile a second time to create conditions
				//_, err = r.Reconcile(request)
				//Expect(err).ToNot(HaveOccurred())

				var endStateNrqlCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, conditionName, &endStateNrqlCondition)
				Expect(err).ToNot(HaveOccurred())
				Expect(endStateNrqlCondition.Spec.Name).To(Equal("NRQL Condition"))

			})

		})

		AfterEach(func() {
			// Delete the policy
			err := k8sClient.Delete(ctx, policy)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("When starting with an existing policy", func() {
		BeforeEach(func() {

			err := k8sClient.Create(ctx, policy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(alertsClient.CreatePolicyCallCount()).To(Equal(1))
			Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, policy)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("and deleting that policy", func() {
			It("should successfully delete", func() {
				err := k8sClient.Delete(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to delete finalizer
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).NotTo(BeNil())

			})
			It("should delete the condition", func() {
				err := k8sClient.Delete(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to delete finalizer
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, conditionName, &endStateCondition)

				Expect(err).To(HaveOccurred())
				Expect(endStateCondition.Spec.Name).ToNot(Equal(policy.Spec.Conditions[0].Spec.Name))

			})
		})
		Context("and New Relic API returns a 404", func() {
			BeforeEach(func() {
				alertsClient.DeletePolicyStub = func(int) (*alerts.Policy, error) {
					return &alerts.Policy{}, errors.New("Imaginary 404 Failure")
				}
			})
			It("should succeed as if a previous reconcile already deleted the policy", func() {
			})
			AfterEach(func() {
				alertsClient.DeletePolicyStub = func(int) (*alerts.Policy, error) {
					return &alerts.Policy{}, nil
				}
				// Delete the policy
				err := k8sClient.Delete(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to delete finalizer
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("When starting with an existing policy", func() {
		BeforeEach(func() {

			err := k8sClient.Create(ctx, policy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(alertsClient.CreatePolicyCallCount()).To(Equal(1))
			Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, policy)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and making no changes ", func() {

			It("should not try to update or create new conditions", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile twice to be fully settled
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(10))

			})

		})

		Context("and updating that policy", func() {
			BeforeEach(func() {
				policy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})

			It("should successfully update", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(1))

			})

		})

		Context("and updating a condition ", func() {
			BeforeEach(func() {
				policy.Spec.Conditions[0].Spec.Name = "New conditionName"
			})

			It("should successfully update", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateCondition nrv1.NrqlAlertCondition               //test-policy1942898816
				err = k8sClient.Get(ctx, conditionName, &endStateCondition) //1942898816
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("New conditionName"))
			})

		})

		Context("and adding another condition ", func() {
			BeforeEach(func() {
				secondConditionSpec := nrv1.NrqlAlertConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     resource.MustParse("30"),
							Operator:     "above",
							Priority:     "critical",
							Threshold:    resource.MustParse("5"),
							TimeFunction: "all",
						},
					},
					Nrql:    nrv1.NrqlQuery{},
					Type:    "",
					Name:    "second alert condition",
					Enabled: true,
				}
				secondCondition := nrv1.NrqlAlertCondition{
					Spec: secondConditionSpec,
					Status: nrv1.NrqlAlertConditionStatus{
						AppliedSpec: &nrv1.NrqlAlertConditionSpec{},
					},
				}

				policy.Spec.Conditions = append(policy.Spec.Conditions, secondCondition)
				conditionName.Name = policy.Name + fmt.Sprintf("%d", nrv1.ComputeHash(&secondConditionSpec))
			})

			It("should add second condition ", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateCondition nrv1.NrqlAlertCondition               //test-policy1468830423
				err = k8sClient.Get(ctx, conditionName, &endStateCondition) //1942898816
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("second alert condition"))
			})

		})

		Context("and when the alerts client returns an error", func() {
			BeforeEach(func() {
				alertsClient.UpdatePolicyStub = func(alerts.Policy) (*alerts.Policy, error) {
					return &alerts.Policy{}, errors.New("oh no")
				}
				policy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})
			It("should return an error", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).To(HaveOccurred())
			})
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, policy)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("When starting with an existing policy with two conditions", func() {
		BeforeEach(func() {

			secondConditionSpec := nrv1.NrqlAlertConditionSpec{
				Terms: []nrv1.AlertConditionTerm{
					{
						Duration:     resource.MustParse("30"),
						Operator:     "above",
						Priority:     "critical",
						Threshold:    resource.MustParse("5"),
						TimeFunction: "all",
					},
				},
				Nrql:    nrv1.NrqlQuery{},
				Type:    "",
				Name:    "second alert condition",
				Enabled: true,
			}
			secondCondition := nrv1.NrqlAlertCondition{
				Spec: secondConditionSpec,
				Status: nrv1.NrqlAlertConditionStatus{
					AppliedSpec: &nrv1.NrqlAlertConditionSpec{},
				},
			}

			policy.Spec.Conditions = append(policy.Spec.Conditions, secondCondition)
			conditionName.Name = policy.Name + fmt.Sprintf("%d", nrv1.ComputeHash(&secondConditionSpec))

			err := k8sClient.Create(ctx, policy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(alertsClient.CreatePolicyCallCount()).To(Equal(1))
			Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, policy)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and removing the second condition ", func() {
			BeforeEach(func() {
				policy.Spec.Conditions = []nrv1.NrqlAlertCondition{policy.Spec.Conditions[0]}

			})

			It("should remove second condition ", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateDeletedCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, conditionName, &endStateDeletedCondition)
				Expect(err).To(HaveOccurred())
				Expect(endStateDeletedCondition.Spec.Name).To(Equal(""))
				conditionName.Name = policy.Name + fmt.Sprintf("%d", nrv1.ComputeHash(conditionSpec))
				var endStateExistingCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, conditionName, &endStateExistingCondition)
				Expect(err).ToNot(HaveOccurred())
				Expect(endStateExistingCondition.Spec.Name).To(Equal("NRQL Condition"))
			})

			It("should not call the alerts API ", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(0))
			})

		})

		Context("and removing the first condition ", func() {
			BeforeEach(func() {
				policy.Spec.Conditions = []nrv1.NrqlAlertCondition{policy.Spec.Conditions[1]}

			})

			It("should remove first condition ", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateDeletedCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, conditionName, &endStateDeletedCondition)
				Expect(err).ToNot(HaveOccurred())
				Expect(endStateDeletedCondition.Spec.Name).To(Equal("second alert condition"))

				var endStateExistingCondition nrv1.NrqlAlertCondition
				conditionName.Name = policy.Name + fmt.Sprintf("%d", nrv1.ComputeHash(conditionSpec))
				err = k8sClient.Get(ctx, conditionName, &endStateExistingCondition)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("nrqlalertconditions.nr.k8s.newrelic.com \"test-policy1991703647\" not found"))
				Expect(endStateExistingCondition.Spec.Name).To(Equal(""))

				var endStatePolicy nrv1.Policy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(endStatePolicy.Spec.Conditions[0].Spec.Name).To(Equal("second alert condition"))
			})

		})

		Context("and when the alerts client returns an error", func() {
			BeforeEach(func() {
				alertsClient.UpdatePolicyStub = func(alerts.Policy) (*alerts.Policy, error) {
					return &alerts.Policy{}, errors.New("oh no")
				}
				policy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})
			It("should return an error", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).To(HaveOccurred())
			})
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, policy)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
