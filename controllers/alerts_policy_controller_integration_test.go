// +build integration

package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	//"github.com/newrelic/newrelic-client-go/pkg/testhelpers"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
	"github.com/newrelic/newrelic-kubernetes-operator/internal/testutil"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestIntegrationAlertsPolicyController_Basic(t *testing.T) {
	t.Parallel()

	policy := testutil.NewTestAlertsPolicy(t)

	// Must come before calling reconciler.Reconcile()
	k8sClient := testutil.AlertsPolicyTestSetup(t)
	err := k8sClient.Create(context.Background(), policy)
	Expect(err).ToNot(HaveOccurred())

	namespacedName := types.NamespacedName{
		Namespace: "default",
		Name:      policy.Name,
	}

	request := ctrl.Request{
		NamespacedName: namespacedName,
	}

	reconciler := &AlertsPolicyReconciler{
		Client:          k8sClient,
		Log:             logf.Log,
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	// TODO
	// otherAlertsClient := alerts.New(testhelpers.NewIntegrationTestConfig(t))

	_, err = reconciler.Reconcile(request)
	assert.NoError(t, err)

	// TODO implement checking that the policy exists.  Where do I get the ID?

	err = k8sClient.Delete(context.Background(), policy)
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(request)
	assert.NoError(t, err)

	// TODO implement checking that the policy has been deleted
}

func TestIntegrationAlertsPolicyController_WithCondition(t *testing.T) {
	t.Parallel()

	policy := testutil.NewTestAlertsPolicy(t)

	// Add a condition to the policy
	conditionSpec := testutil.NewTestAlertsNrqlConditionSpec(t)
	policy.Spec.Conditions = append(policy.Spec.Conditions, nrv1.AlertsPolicyCondition{Spec: *conditionSpec})

	// Must come before calling reconciler.Reconcile()
	k8sClient := testutil.AlertsPolicyTestSetup(t)

	err := k8sClient.Create(context.Background(), policy)
	Expect(err).ToNot(HaveOccurred())

	namespacedName := types.NamespacedName{
		Namespace: "default",
		Name:      policy.Name,
	}

	request := ctrl.Request{
		NamespacedName: namespacedName,
	}

	reconciler := &AlertsPolicyReconciler{
		Client:          k8sClient,
		Log:             logf.Log,
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	_, err = reconciler.Reconcile(request)
	assert.NoError(t, err)

	// TODO
	// assert the policy exists

	err = k8sClient.Delete(context.Background(), policy)
	assert.NoError(t, err)

	// TODO
	// assert the policy does not exist

	_, err = reconciler.Reconcile(request)
	assert.NoError(t, err)
}

var _ = Describe("policy reconciliation", func() {
	var (
		ctx            context.Context
		r              *AlertsPolicyReconciler
		t              *testing.T
		policy         *nrv1.AlertsPolicy
		conditionSpec  *nrv1.AlertsNrqlConditionSpec
		request        ctrl.Request
		namespacedName types.NamespacedName
		conditionName  types.NamespacedName
		//expectedEvents []string
		//secret        *v1.Secret
		// fakeAlertFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
		deletedConditionNamespace types.NamespacedName
		k8sClient                 client.Client
		// otherAlertsClient         alerts.Alerts
		alertsClient interfacesfakes.FakeNewRelicAlertsClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		t = &testing.T{}
		k8sClient = testutil.AlertsPolicyTestSetup(t)
		alertsClient = interfacesfakes.FakeNewRelicAlertsClient{}

		policy = testutil.NewTestAlertsPolicy(t)

		// otherAlertsClient = alerts.New(testhelpers.NewIntegrationTestConfig(t))

		// Add a condition to the policy
		conditionSpec = testutil.NewTestAlertsNrqlConditionSpec(t)
		policy.Spec.Conditions = append(policy.Spec.Conditions, nrv1.AlertsPolicyCondition{Spec: *conditionSpec})

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      policy.Name,
		}

		request = ctrl.Request{
			NamespacedName: namespacedName,
		}

		r = &AlertsPolicyReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: interfaces.InitializeAlertsClient,
		}
	})

	Context("When starting with no policies", func() {
		Context("when creating a valid policy", func() {
			It("should create that policy", func() {
				err := k8sClient.Create(ctx, policy)
				Expect(err).To(BeNil())

				_, err = r.Reconcile(request)
				Expect(err).To(BeNil())
			})

			It("updates the policyID on the AlertsPolicy resource", func() {
				err := k8sClient.Create(ctx, policy)
				Expect(err).To(BeNil())

				_, err = r.Reconcile(request)
				Expect(err).To(BeNil())

				var endStatePolicy nrv1.AlertsPolicy

				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())

				searchResults, err := alertsClient.QueryPolicy(policy.Spec.AccountID, endStatePolicy.Status.PolicyID)
				Expect(err).To(BeNil())
				Expect(searchResults.ID).To(Equal(endStatePolicy.Status.PolicyID))
			})

			It("creates the condition with attributes from the AlertsPolicy", func() {
				err := k8sClient.Create(ctx, policy)
				Expect(err).To(BeNil())

				_, err = r.Reconcile(request)
				Expect(err).To(BeNil())

				// Get the policy from k8s
				var endStatePolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())

				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}

				// Get the condition from k8s
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
			})

			It("creates the condition with inherited attributes from the Policy resource", func() {

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())

				Expect(endStateCondition.Spec.ExistingPolicyID).To(Equal(333))
				Expect(endStateCondition.Spec.Region).To(Equal(policy.Spec.Region))
				Expect(endStateCondition.Spec.APIKey).To(Equal(policy.Spec.APIKey))

			})
		})

		Context("when the New Relic API returns an error", func() {
			BeforeEach(func() {
				alertsClient.CreatePolicyMutationStub = func(int, alerts.AlertsPolicyInput) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("Any Error Goes Here")
				}
			})

			It("should not update the PolicyID", func() {

				createErr := k8sClient.Create(ctx, policy)
				Expect(createErr).ToNot(HaveOccurred())

				// call reconcile
				_, reconcileErr := r.Reconcile(request)
				Expect(reconcileErr).To(HaveOccurred())
				Expect(reconcileErr.Error()).To(Equal("Any Error Goes Here"))

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

				Expect(alertsClient.CreatePolicyMutationCallCount()).To(Equal(1))

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("NRQL Condition"))

			})

			Context("when creating a valid policy with conditions with k8 resource name set", func() {
				It("should create the conditions with an auto-generated name ignoring the manual name", func() {

					policy.Spec.Conditions[0].Name = "my custom name"

					err := k8sClient.Create(ctx, policy)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreatePolicyMutationCallCount()).To(Equal(1))

					var endStatePolicy nrv1.Policy
					var endStateCondition nrv1.NrqlAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
					Expect(err).To(BeNil())
					conditionNameType := types.NamespacedName{
						Name:      endStatePolicy.Spec.Conditions[0].Name,
						Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
					}
					err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Spec.Name).To(Equal("NRQL Condition"))
					Expect(endStateCondition.Name).ToNot(Equal("my custom name"))
				})
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

			Expect(alertsClient.CreatePolicyMutationCallCount()).To(Equal(1))
			Expect(alertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

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
				alertsClient.DeletePolicyMutationStub = func(int, string) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("Imaginary 404 Failure")
				}
			})
			It("should succeed as if a previous reconcile already deleted the policy", func() {
			})
			AfterEach(func() {
				alertsClient.DeletePolicyMutationStub = func(int, string) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, nil
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

			Expect(alertsClient.CreatePolicyMutationCallCount()).To(Equal(1))
			Expect(alertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, policy)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and making no changes ", func() {

			It("should not try to update or create new conditions", func() {
				initialConditionName := policy.Spec.Conditions[0].Name
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile twice to be fully settled
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(alertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())

				Expect(endStatePolicy.Status.AppliedSpec.Conditions[0].Name).To(Equal(initialConditionName))
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Status.AppliedSpec.Conditions[0].Name,
					Namespace: endStatePolicy.Status.AppliedSpec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Name).To(Equal(initialConditionName))
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
				Expect(alertsClient.UpdatePolicyMutationCallCount()).To(Equal(1))

			})

		})

		Context("and updating a condition name", func() {
			BeforeEach(func() {
				//clear out the Name and Namespace since those aren't stored in yaml so are blank when applying yaml
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""
				policy.Spec.Conditions[0].Spec.Name = "New conditionName"
			})

			It("should create a new condition with the new name", func() {
				originalConditionName := policy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("New conditionName"))
				Expect(endStateCondition.Name).ToNot(Equal(originalConditionName))
			})

			It("should delete the old condition ", func() {
				originalConditionName := policy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())

				var originalCondition nrv1.NrqlAlertCondition
				originalConditionNamespaceType := types.NamespacedName{
					Name:      originalConditionName,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, originalConditionNamespaceType, &originalCondition)
				Expect(err).ToNot(BeNil())
				Expect(originalCondition.Spec.Name).To(Equal(""))
			})

		})

		Context("and updating a condition ", func() {
			BeforeEach(func() {
				//clear out the Name and Namespace since those aren't stored in yaml so are blank when appling yaml
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""
				policy.Spec.Conditions[0].Spec.Nrql.Query = "SELECT count(*) FROM MyEvent"
			})

			It("should update the existing NrqlAlertCondition with the updated spec", func() {
				originalConditionName := policy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Nrql.Query).To(Equal("SELECT count(*) FROM MyEvent"))
				Expect(endStateCondition.Name).To(Equal(originalConditionName))

			})

			It("should set the inherited values on the updated condition", func() {
				originalConditionName := policy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Nrql.Query).To(Equal("SELECT count(*) FROM MyEvent"))
				Expect(endStateCondition.Name).To(Equal(originalConditionName))
				Expect(endStateCondition.Spec.Region).To(Equal("us"))
				Expect(endStateCondition.Spec.APIKey).To(Equal("112233"))

			})

		})

		Context("and adding another condition ", func() {
			BeforeEach(func() {
				secondConditionSpec := nrv1.AlertsNrqlConditionSpec{
					Terms: []nrv1.AlertsNrqlConditionTerm{
						{
							ThresholdDuration:    30,
							Operator:             "above",
							Priority:             "critical",
							Threshold:            "5",
							ThresholdOccurrences: alerts.ThresholdOccurrences.All,
						},
					},
					Nrql:    alerts.NrqlConditionQuery{},
					Type:    "",
					Name:    "second alert condition",
					Enabled: true,
				}
				secondCondition := nrv1.AlertsPolicyCondition{
					Spec: secondConditionSpec,
				}

				policy.Spec.Conditions = append(policy.Spec.Conditions, secondCondition)
			})

			It("should add second condition ", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.AlertsPolicy //test-policy1942898816
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy) //1942898816
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[1].Name,
					Namespace: endStatePolicy.Spec.Conditions[1].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("second alert condition"))
				Expect(endStateCondition.Spec.Region).To(Equal("us"))
				Expect(endStateCondition.Spec.APIKey).To(Equal("112233"))
			})

		})

		Context("and when the alerts client returns an error", func() {
			BeforeEach(func() {
				alertsClient.UpdatePolicyMutationStub = func(int, string, alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("oh no")
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

			secondConditionSpec := nrv1.AlertsNrqlConditionSpec{
				Terms: []nrv1.AlertsNrqlConditionTerm{
					{
						ThresholdDuration:    30,
						Operator:             "above",
						Priority:             "critical",
						Threshold:            "5",
						ThresholdOccurrences: alerts.ThresholdOccurrences.All,
					},
				},
				Nrql:    alerts.NrqlConditionQuery{},
				Type:    "",
				Name:    "second alert condition",
				Enabled: true,
			}
			secondCondition := nrv1.AlertsPolicyCondition{
				Spec: secondConditionSpec,
			}

			policy.Spec.Conditions = append(policy.Spec.Conditions, secondCondition)

			err := k8sClient.Create(ctx, policy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(alertsClient.CreatePolicyMutationCallCount()).To(Equal(1))
			Expect(alertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, policy)
			Expect(err).ToNot(HaveOccurred())

			deletedConditionNamespace = types.NamespacedName{
				Namespace: policy.Spec.Conditions[1].Namespace,
				Name:      policy.Spec.Conditions[1].Name,
			}
		})

		Context("and removing the second condition ", func() {
			BeforeEach(func() {

				policy.Spec.Conditions = []nrv1.AlertsPolicyCondition{policy.Spec.Conditions[0]}

			})

			It("should remove second condition ", func() {
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("NRQL Condition"))
				Expect(len(endStatePolicy.Spec.Conditions)).To(Equal(1))
				var deletedCondition nrv1.AlertsNrqlCondition
				Expect(deletedConditionNamespace.Name).ToNot(Equal(endStateCondition.Name))
				err = k8sClient.Get(ctx, deletedConditionNamespace, &deletedCondition)
				Expect(err).ToNot(BeNil())
				Expect(deletedCondition.Name).To(Equal(""))

			})

			It("should not call the alerts API ", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(alertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))
			})

		})

		Context("and removing the first condition ", func() {
			BeforeEach(func() {
				policy.Spec.Conditions = []nrv1.AlertsPolicyCondition{policy.Spec.Conditions[1]}

			})

			It("should remove first condition ", func() {
				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy //test-policy1942898816
				var endStateCondition nrv1.NrqlAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy) //1942898816
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("second alert condition"))
			})

		})

		Context("and when the alerts client returns an error", func() {
			BeforeEach(func() {
				alertsClient.UpdatePolicyMutationStub = func(int, string, alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("oh no")
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
