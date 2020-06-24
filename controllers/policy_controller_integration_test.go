// +build integration

package controllers

import (
	"context"
	"errors"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
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
		ctx                       context.Context
		r                         *PolicyReconciler
		policy                    *nrv1.Policy
		conditionSpec             *nrv1.ConditionSpec
		request                   ctrl.Request
		namespacedName            types.NamespacedName
		conditionName             types.NamespacedName
		fakeAlertFunc             func(string, string) (interfaces.NewRelicAlertsClient, error)
		deletedConditionNamespace types.NamespacedName
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

		newRelicAgent := newrelic.Application{}

		r = &PolicyReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
			NewRelicAgent:   newRelicAgent,
		}

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "test-policy",
		}

		request = ctrl.Request{NamespacedName: namespacedName}
	})

	Context("When starting with no policies", func() {
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
				NewRelicAgent:   newrelic.Application{},
			}

			conditionSpec = &nrv1.ConditionSpec{
				nrv1.GenericConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "5",
							TimeFunction: "all",
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				nrv1.NrqlSpecificSpec{
					Nrql: nrv1.NrqlQuery{
						Query:      "SELECT 1 FROM MyEvents",
						SinceValue: "5",
					},
					ValueFunction:       "max",
					ViolationCloseTimer: 60,
					ExpectedGroups:      2,
					IgnoreOverlap:       true,
				},
				nrv1.APMSpecificSpec{},
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
					Region:             "us",
					Conditions: []nrv1.PolicyCondition{
						{
							Spec: *conditionSpec,
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

			request = ctrl.Request{NamespacedName: namespacedName}
		})

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

			It("creates the NRQL condition with attributes from the Policy", func() {
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
				Expect(endStateCondition.Spec.Nrql.Query).To(Equal("SELECT 1 FROM MyEvents"))
				Expect(endStateCondition.Spec.Terms[0].Priority).To(Equal("critical"))
				Expect(endStateCondition.Spec.Enabled).To(BeTrue())
			})

			It("creates the NRQL condition with inherited attributes from the Policy resource", func() {
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
				Expect(reconcileErr.Error()).To(Equal("Any Error Goes Here"))

				var endStatePolicy nrv1.Policy
				getErr := k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(getErr).ToNot(HaveOccurred())
				Expect(endStatePolicy.Status.PolicyID).To(Equal(0))
			})
		})

		Context("when creating a valid policy with apm conditions", func() {
			It("should create the conditions", func() {
				conditionSpec = &nrv1.ConditionSpec{
					GenericConditionSpec: nrv1.GenericConditionSpec{
						Terms: []nrv1.AlertConditionTerm{
							{
								Duration:     "30",
								Operator:     "above",
								Priority:     "critical",
								Threshold:    "1.5",
								TimeFunction: "all",
							},
						},
						Type:       "apm_app_metric",
						Name:       "APM Condition",
						RunbookURL: "http://test.com/runbook",
						Enabled:    true,
					},
					NrqlSpecificSpec: nrv1.NrqlSpecificSpec{},
					APMSpecificSpec: nrv1.APMSpecificSpec{
						Metric:              "Custom/foo",
						UserDefined:         alerts.ConditionUserDefined{},
						Scope:               "application",
						Entities:            []string{"333"},
						GCMetric:            "",
						ViolationCloseTimer: 60,
					},
				}

				policy.Spec.Conditions[0] = nrv1.PolicyCondition{
					Spec: *conditionSpec,
				}

				err := k8sClient.Create(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(alertsClient.CreatePolicyCallCount()).To(Equal(1))

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.ApmAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("APM Condition"))
			})

			Context("when creating a valid policy with conditions with k8 resource name set", func() {
				It("should create the conditions with an auto-generated name ignoring the manual name", func() {
					policy.Spec.Conditions[0].Name = "my custom name"

					err := k8sClient.Create(ctx, policy)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreatePolicyCallCount()).To(Equal(1))

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

	Context("When starting with an existing policy with a NRQL condition", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.ConditionSpec{
				nrv1.GenericConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "5",
							TimeFunction: "all",
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				nrv1.NrqlSpecificSpec{
					Nrql: nrv1.NrqlQuery{
						Query:      "SELECT 1 FROM MyEvents",
						SinceValue: "5",
					},
					ValueFunction:       "max",
					ViolationCloseTimer: 60,
					ExpectedGroups:      2,
					IgnoreOverlap:       true,
				},
				nrv1.APMSpecificSpec{},
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
					Region:             "us",
					Conditions: []nrv1.PolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.PolicyStatus{
					AppliedSpec: &nrv1.PolicySpec{},
					PolicyID:    0,
				},
			}

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

	Context("When starting with an existing policy with a NRQL condition", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.ConditionSpec{
				nrv1.GenericConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "5",
							TimeFunction: "all",
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				nrv1.NrqlSpecificSpec{
					Nrql: nrv1.NrqlQuery{
						Query:      "SELECT 1 FROM MyEvents",
						SinceValue: "5",
					},
					ValueFunction:       "max",
					ViolationCloseTimer: 60,
					ExpectedGroups:      2,
					IgnoreOverlap:       true,
				},
				nrv1.APMSpecificSpec{},
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
					Region:             "us",
					Conditions: []nrv1.PolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.PolicyStatus{
					AppliedSpec: &nrv1.PolicySpec{},
					PolicyID:    0,
				},
			}

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
				initialConditionName := policy.Spec.Conditions[0].Name
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile twice to be fully settled
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(0))

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
				Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(1))
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
				secondConditionSpec := nrv1.ConditionSpec{
					nrv1.GenericConditionSpec{
						Terms: []nrv1.AlertConditionTerm{
							{
								Duration:     "30",
								Operator:     "above",
								Priority:     "critical",
								Threshold:    "5",
								TimeFunction: "all",
							},
						},
						Type:       "NRQL",
						Name:       "second alert condition",
						RunbookURL: "http://test.com/runbook",
						Enabled:    true,
					},
					nrv1.NrqlSpecificSpec{
						Nrql: nrv1.NrqlQuery{
							Query:      "SELECT 1 FROM MyEvents",
							SinceValue: "5",
						},
						ValueFunction:       "max",
						ViolationCloseTimer: 60,
						ExpectedGroups:      2,
						IgnoreOverlap:       true,
					},
					nrv1.APMSpecificSpec{},
				}
				secondCondition := nrv1.PolicyCondition{
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

				var endStatePolicy nrv1.Policy //test-policy1942898816
				var endStateCondition nrv1.NrqlAlertCondition
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

	Context("When starting with an existing policy with an APM condition", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.ConditionSpec{

				GenericConditionSpec: nrv1.GenericConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "1.5",
							TimeFunction: "all",
						},
					},
					Type:       "apm_app_metric",
					Name:       "APM Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				NrqlSpecificSpec: nrv1.NrqlSpecificSpec{},
				APMSpecificSpec: nrv1.APMSpecificSpec{
					Metric:              "Custom/foo",
					UserDefined:         alerts.ConditionUserDefined{},
					Scope:               "application",
					Entities:            []string{"333"},
					GCMetric:            "",
					ViolationCloseTimer: 60,
				},
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
					Region:             "us",
					Conditions: []nrv1.PolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.PolicyStatus{
					AppliedSpec: &nrv1.PolicySpec{},
					PolicyID:    0,
				},
			}

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
				initialConditionName := policy.Spec.Conditions[0].Name
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile twice to be fully settled
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(0))

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.ApmAlertCondition
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
				Expect(endStateCondition.Spec.Name).To(Equal("APM Condition"))
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

		Context("and updating a condition name", func() {
			BeforeEach(func() {
				//clear out the Name and Namespace since those aren't stored in yaml so are blank when applying yaml
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""
				policy.Spec.Conditions[0].Spec.Name = "New conditionName"
			})

			It("should create a new apm condition with the new name", func() {
				originalConditionName := policy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.ApmAlertCondition
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

				var originalCondition nrv1.ApmAlertCondition
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
				policy.Spec.Conditions[0].Spec.Metric = "Custom/bar"
			})

			It("should update the existing ApmAlertCondition with the updated spec", func() {
				originalConditionName := policy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, policy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStatePolicy nrv1.Policy
				var endStateCondition nrv1.ApmAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Metric).To(Equal("Custom/bar"))
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
				var endStateCondition nrv1.ApmAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[0].Name,
					Namespace: endStatePolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Name).To(Equal(originalConditionName))
				Expect(endStateCondition.Spec.Region).To(Equal("us"))
				Expect(endStateCondition.Spec.APIKey).To(Equal("112233"))
			})
		})

		Context("and adding another apm condition ", func() {
			BeforeEach(func() {
				secondConditionSpec := nrv1.ConditionSpec{
					GenericConditionSpec: nrv1.GenericConditionSpec{
						Terms: []nrv1.AlertConditionTerm{
							{
								Duration:     "30",
								Operator:     "above",
								Priority:     "critical",
								Threshold:    "1.5",
								TimeFunction: "all",
							},
						},
						Type:       "apm_app_metric",
						Name:       "Second APM Condition",
						RunbookURL: "http://test.com/runbook",
						Enabled:    true,
					},
					NrqlSpecificSpec: nrv1.NrqlSpecificSpec{},
					APMSpecificSpec: nrv1.APMSpecificSpec{
						Metric:              "Custom/foo",
						UserDefined:         alerts.ConditionUserDefined{},
						Scope:               "application",
						Entities:            []string{"333"},
						GCMetric:            "",
						ViolationCloseTimer: 60,
					},
				}

				secondCondition := nrv1.PolicyCondition{
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

				var endStatePolicy nrv1.Policy //test-policy1942898816
				var endStateCondition nrv1.ApmAlertCondition
				err = k8sClient.Get(ctx, namespacedName, &endStatePolicy) //1942898816
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStatePolicy.Spec.Conditions[1].Name,
					Namespace: endStatePolicy.Spec.Conditions[1].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("Second APM Condition"))
				Expect(endStateCondition.Spec.Region).To(Equal("us"))
				Expect(endStateCondition.Spec.APIKey).To(Equal("112233"))
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

	Context("When starting with an existing policy with two NRQL conditions", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.ConditionSpec{
				nrv1.GenericConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "5",
							TimeFunction: "all",
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				nrv1.NrqlSpecificSpec{
					Nrql: nrv1.NrqlQuery{
						Query:      "SELECT 1 FROM MyEvents",
						SinceValue: "5",
					},
					ValueFunction:       "max",
					ViolationCloseTimer: 60,
					ExpectedGroups:      2,
					IgnoreOverlap:       true,
				},
				nrv1.APMSpecificSpec{},
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
					Region:             "us",
					Conditions: []nrv1.PolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.PolicyStatus{
					AppliedSpec: &nrv1.PolicySpec{},
					PolicyID:    0,
				},
			}

			secondConditionSpec := nrv1.ConditionSpec{
				nrv1.GenericConditionSpec{
					Terms: []nrv1.AlertConditionTerm{
						{
							Duration:     "30",
							Operator:     "above",
							Priority:     "critical",
							Threshold:    "5",
							TimeFunction: "all",
						},
					},
					Type:       "NRQL",
					Name:       "second alert condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				nrv1.NrqlSpecificSpec{
					Nrql: nrv1.NrqlQuery{
						Query:      "SELECT 1 FROM MyEvents",
						SinceValue: "5",
					},
					ValueFunction:       "max",
					ViolationCloseTimer: 60,
					ExpectedGroups:      2,
					IgnoreOverlap:       true,
				},
				nrv1.APMSpecificSpec{},
			}
			secondCondition := nrv1.PolicyCondition{
				Spec: secondConditionSpec,
			}

			policy.Spec.Conditions = append(policy.Spec.Conditions, secondCondition)

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

			deletedConditionNamespace = types.NamespacedName{
				Namespace: policy.Spec.Conditions[1].Namespace,
				Name:      policy.Spec.Conditions[1].Name,
			}
		})

		Context("and removing the second condition ", func() {
			BeforeEach(func() {
				policy.Spec.Conditions = []nrv1.PolicyCondition{policy.Spec.Conditions[0]}
			})

			It("should remove second condition ", func() {
				policy.Spec.Conditions[0].Name = ""
				policy.Spec.Conditions[0].Namespace = ""

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
				Expect(endStateCondition.Spec.Name).To(Equal("NRQL Condition"))
				Expect(len(endStatePolicy.Spec.Conditions)).To(Equal(1))
				var deletedCondition nrv1.NrqlAlertCondition
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
				Expect(alertsClient.UpdatePolicyCallCount()).To(Equal(0))
			})
		})

		Context("and removing the first condition ", func() {
			BeforeEach(func() {
				policy.Spec.Conditions = []nrv1.PolicyCondition{policy.Spec.Conditions[1]}

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
