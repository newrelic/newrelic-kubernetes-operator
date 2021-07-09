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

var _ = Describe("alertspolicy reconciliation", func() {
	var (
		ctx                       context.Context
		r                         *AlertsPolicyReconciler
		alertspolicy              *nrv1.AlertsPolicy
		conditionSpec             *nrv1.AlertsPolicyConditionSpec
		request                   ctrl.Request
		namespacedName            types.NamespacedName
		conditionName             types.NamespacedName
		fakeAlertFunc             func(string, string) (interfaces.NewRelicAlertsClient, error)
		deletedConditionNamespace types.NamespacedName
		mockAlertsClient          interfacesfakes.FakeNewRelicAlertsClient
	)

	BeforeEach(func() {
		ctx = context.Background()

		mockAlertsClient = interfacesfakes.FakeNewRelicAlertsClient{}

		fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return &mockAlertsClient, nil
		}

		mockAlertsClient.CreatePolicyMutationStub = func(accountID int, a alerts.AlertsPolicyInput) (*alerts.AlertsPolicy, error) {
			return &alerts.AlertsPolicy{
				ID: "333",
			}, nil
		}

		mockAlertsClient.UpdatePolicyMutationStub = func(int, string, alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error) {
			return &alerts.AlertsPolicy{
				ID: "222",
			}, nil
		}

		mockAlertsClient.UpdatePolicyChannelsStub = func(policy int, channels []int) (*alerts.PolicyChannels, error) {
			return &alerts.PolicyChannels{
				ID:         policy,
				ChannelIDs: channels,
			}, nil
		}

		mockAlertsClient.DeletePolicyChannelStub = func(policy int, channel int) (*alerts.Channel, error) {
			return &alerts.Channel{
				ID: policy,
			}, nil
		}

		newRelicAgent := newrelic.Application{}

		r = &AlertsPolicyReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
			NewRelicAgent:   newRelicAgent,
		}

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "test-alertspolicy",
		}
		request = ctrl.Request{NamespacedName: namespacedName}
	})

	Context("When starting with no policies", func() {
		BeforeEach(func() {
			ctx = context.Background()

			mockAlertsClient = interfacesfakes.FakeNewRelicAlertsClient{}

			fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
				return &mockAlertsClient, nil
			}

			mockAlertsClient.CreatePolicyMutationStub = func(accountID int, a alerts.AlertsPolicyInput) (*alerts.AlertsPolicy, error) {
				return &alerts.AlertsPolicy{
					ID: "333",
				}, nil
			}

			mockAlertsClient.UpdatePolicyMutationStub = func(int, string, alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error) {
				return &alerts.AlertsPolicy{
					ID: "222",
				}, nil
			}

			r = &AlertsPolicyReconciler{
				Client:          k8sClient,
				Log:             logf.Log,
				AlertClientFunc: fakeAlertFunc,
				NewRelicAgent:   newrelic.Application{},
			}

			conditionSpec = &nrv1.AlertsPolicyConditionSpec{
				AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
					Terms: []nrv1.AlertsNrqlConditionTerm{
						{
							ThresholdDuration:    30,
							Operator:             "above",
							Priority:             "critical",
							Threshold:            "5",
							ThresholdOccurrences: alerts.ThresholdOccurrences.All,
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{
					Nrql: alerts.NrqlConditionQuery{
						Query:            "SELECT 1 FROM MyEvents",
						EvaluationOffset: 5,
					},
					ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
					ViolationTimeLimit: "60",
					ExpectedGroups:     2,
					IgnoreOverlap:      true,
				},
				AlertsAPMSpecificSpec:      nrv1.AlertsAPMSpecificSpec{},
				AlertsBaselineSpecificSpec: nrv1.AlertsBaselineSpecificSpec{},
			}

			alertspolicy = &nrv1.AlertsPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-alertspolicy",
					Namespace: "default",
				},
				Spec: nrv1.AlertsPolicySpec{
					Name:               "test alertspolicy",
					APIKey:             "112233",
					IncidentPreference: "PER_POLICY",
					Region:             "us",
					ChannelIDs:         []int{1, 2},
					Conditions: []nrv1.AlertsPolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},

				Status: nrv1.AlertsPolicyStatus{
					AppliedSpec: &nrv1.AlertsPolicySpec{},
					PolicyID:    "",
				},
			}

			namespacedName = types.NamespacedName{
				Namespace: "default",
				Name:      "test-alertspolicy",
			}
			request = ctrl.Request{NamespacedName: namespacedName}
		})

		Context("when creating a valid alertspolicy", func() {
			It("should create that alertspolicy", func() {

				err := k8sClient.Create(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

			})

			It("updates the Policy resource", func() {

				err := k8sClient.Create(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				Expect(endStateAlertsPolicy.Status.PolicyID).To(Equal("333"))

			})

			It("creates the NRQL condition with attributes from the AlertsPolicy", func() {
				err := k8sClient.Create(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Nrql.Query).To(Equal("SELECT 1 FROM MyEvents"))
				Expect(string(endStateCondition.Spec.Terms[0].Priority)).To(Equal("critical"))
				Expect(endStateCondition.Spec.Enabled).To(BeTrue())
			})

			It("creates ownership reference to parent alerts policy", func() {
				err := k8sClient.Create(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.ObjectMeta.OwnerReferences[0].Name).To(Equal(endStateAlertsPolicy.Name))
			})

			It("creates the NRQL condition with inherited attributes from the AlertsPolicy resource", func() {
				err := k8sClient.Create(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())

				Expect(endStateCondition.Spec.ExistingPolicyID).To(Equal("333"))
				Expect(endStateCondition.Spec.Region).To(Equal(alertspolicy.Spec.Region))
				Expect(endStateCondition.Spec.APIKey).To(Equal(alertspolicy.Spec.APIKey))
			})

			It("adds expected alertsChannels", func() {
				err := k8sClient.Create(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockAlertsClient.UpdatePolicyChannelsCallCount()).To(Equal(1))

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				Expect(endStateAlertsPolicy.Status.AppliedSpec.ChannelIDs).To(Equal([]int{1, 2}))

			})
		})

		Context("when the New Relic API returns an error", func() {
			BeforeEach(func() {
				mockAlertsClient.CreatePolicyMutationStub = func(int, alerts.AlertsPolicyInput) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("any Error Goes Here")
				}
			})

			It("should not update the AlertsPolicyID", func() {
				createErr := k8sClient.Create(ctx, alertspolicy)
				Expect(createErr).ToNot(HaveOccurred())

				// call reconcile
				_, reconcileErr := r.Reconcile(request)
				Expect(reconcileErr).To(HaveOccurred())
				Expect(reconcileErr.Error()).To(Equal("any Error Goes Here"))

				var endStateAlertsPolicy nrv1.AlertsPolicy
				getErr := k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(getErr).ToNot(HaveOccurred())
				Expect(endStateAlertsPolicy.Status.PolicyID).To(Equal(""))
			})
		})

		Context("when creating a valid alertspolicy with apm conditions", func() {
			It("should create the conditions", func() {
				conditionSpec = &nrv1.AlertsPolicyConditionSpec{
					AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
						Terms: []nrv1.AlertsNrqlConditionTerm{
							{
								ThresholdDuration:    30,
								Operator:             "above",
								Priority:             "critical",
								Threshold:            "1.5",
								ThresholdOccurrences: alerts.ThresholdOccurrences.All,
							},
						},
						Type:       "apm_app_metric",
						Name:       "APM Condition",
						RunbookURL: "http://test.com/runbook",
						Enabled:    true,
					},
					AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{},
					AlertsAPMSpecificSpec: nrv1.AlertsAPMSpecificSpec{
						Metric:              "Custom/foo",
						UserDefined:         alerts.ConditionUserDefined{},
						Scope:               "application",
						Entities:            []string{"333"},
						GCMetric:            "",
						ViolationCloseTimer: 60,
					},
				}
				alertspolicy.Spec.Conditions[0] = nrv1.AlertsPolicyCondition{
					Spec: *conditionSpec,
				}

				err := k8sClient.Create(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockAlertsClient.CreatePolicyMutationCallCount()).To(Equal(1))

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsAPMCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("APM Condition"))
			})

			Context("when creating a valid alertspolicy with conditions with k8 resource name set", func() {
				It("should create the conditions with an auto-generated name ignoring the manual name", func() {
					alertspolicy.Spec.Conditions[0].Name = "my custom name"

					err := k8sClient.Create(ctx, alertspolicy)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(mockAlertsClient.CreatePolicyMutationCallCount()).To(Equal(1))

					var endStateAlertsPolicy nrv1.AlertsPolicy
					var endStateCondition nrv1.AlertsNrqlCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
					Expect(err).To(BeNil())
					conditionNameType := types.NamespacedName{
						Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
						Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
					}
					err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Spec.Name).To(Equal("NRQL Condition"))
					Expect(endStateCondition.Name).ToNot(Equal("my custom name"))
				})
			})
		})

		AfterEach(func() {
			// Delete the alertspolicy
			err := k8sClient.Delete(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When starting with an existing alertspolicy with a NRQL condition", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.AlertsPolicyConditionSpec{
				AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
					Terms: []nrv1.AlertsNrqlConditionTerm{
						{
							ThresholdDuration:    30,
							Operator:             "above",
							Priority:             "critical",
							Threshold:            "5",
							ThresholdOccurrences: alerts.ThresholdOccurrences.All,
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{
					Nrql: alerts.NrqlConditionQuery{
						Query:            "SELECT 1 FROM MyEvents",
						EvaluationOffset: 5,
					},
					ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
					ViolationTimeLimit: "60",
					ExpectedGroups:     2,
					IgnoreOverlap:      true,
				},
				AlertsAPMSpecificSpec:      nrv1.AlertsAPMSpecificSpec{},
				AlertsBaselineSpecificSpec: nrv1.AlertsBaselineSpecificSpec{},
			}

			alertspolicy = &nrv1.AlertsPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-alertspolicy",
					Namespace: "default",
				},
				Spec: nrv1.AlertsPolicySpec{
					Name:               "test alertspolicy",
					APIKey:             "112233",
					IncidentPreference: "PER_POLICY",
					Region:             "us",
					ChannelIDs:         []int{1, 2},
					Conditions: []nrv1.AlertsPolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.AlertsPolicyStatus{
					AppliedSpec: &nrv1.AlertsPolicySpec{},
					PolicyID:    "",
				},
			}

			err := k8sClient.Create(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockAlertsClient.CreatePolicyMutationCallCount()).To(Equal(1))
			Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, alertspolicy)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and deleting that alertspolicy", func() {
			It("should successfully delete", func() {
				err := k8sClient.Delete(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to delete finalizer
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).NotTo(BeNil())
			})

			It("should delete the condition", func() {
				err := k8sClient.Delete(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to delete finalizer
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, conditionName, &endStateCondition)

				Expect(err).To(HaveOccurred())
				Expect(endStateCondition.Spec.Name).ToNot(Equal(alertspolicy.Spec.Conditions[0].Spec.Name))
			})
		})

		Context("and New Relic API returns a 404", func() {
			BeforeEach(func() {
				mockAlertsClient.DeletePolicyMutationStub = func(int, string) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("imaginary 404 Failure")
				}
			})

			It("should succeed as if a previous reconcile already deleted the alertspolicy", func() {
			})

			AfterEach(func() {
				mockAlertsClient.DeletePolicyMutationStub = func(int, string) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, nil
				}
				// Delete the alertspolicy
				err := k8sClient.Delete(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to delete finalizer
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("and updating the list of Alerts channnels", func() {
			BeforeEach(func() {
				alertspolicy.Spec.ChannelIDs = []int{1, 3}
			})

			It("should call the New Relic Alerts API", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockAlertsClient.UpdatePolicyChannelsCallCount()).To(Equal(2))
				policyID, alertsChannels := mockAlertsClient.UpdatePolicyChannelsArgsForCall(1)
				Expect(policyID).To(Equal(333))
				Expect(alertsChannels).To(Equal([]int{3}))

				Expect(mockAlertsClient.DeletePolicyChannelCallCount()).To(Equal(1))
				_, deletedChannnel := mockAlertsClient.DeletePolicyChannelArgsForCall(0)
				Expect(deletedChannnel).To(Equal(2))

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				Expect(endStateAlertsPolicy.Status.AppliedSpec.ChannelIDs).To(Equal([]int{1, 3}))
			})

			AfterEach(func() {
				mockAlertsClient.DeletePolicyMutationStub = func(int, string) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, nil
				}

				// Delete the alertspolicy
				err := k8sClient.Delete(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to delete finalizer
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("When starting with an existing alertspolicy with a NRQL condition", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.AlertsPolicyConditionSpec{
				AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
					Terms: []nrv1.AlertsNrqlConditionTerm{
						{
							ThresholdDuration:    30,
							Operator:             "above",
							Priority:             "critical",
							Threshold:            "5",
							ThresholdOccurrences: alerts.ThresholdOccurrences.All,
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{
					Nrql: alerts.NrqlConditionQuery{
						Query:            "SELECT 1 FROM MyEvents",
						EvaluationOffset: 5,
					},
					ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
					ViolationTimeLimit: "60",
					ExpectedGroups:     2,
					IgnoreOverlap:      true,
				},
				AlertsAPMSpecificSpec:      nrv1.AlertsAPMSpecificSpec{},
				AlertsBaselineSpecificSpec: nrv1.AlertsBaselineSpecificSpec{},
			}

			alertspolicy = &nrv1.AlertsPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-alertspolicy",
					Namespace: "default",
				},
				Spec: nrv1.AlertsPolicySpec{
					Name:               "test alertspolicy",
					APIKey:             "112233",
					IncidentPreference: "PER_POLICY",
					Region:             "us",
					Conditions: []nrv1.AlertsPolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.AlertsPolicyStatus{
					AppliedSpec: &nrv1.AlertsPolicySpec{},
					PolicyID:    "",
				},
			}

			err := k8sClient.Create(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockAlertsClient.CreatePolicyMutationCallCount()).To(Equal(1))
			Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, alertspolicy)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and making no changes ", func() {
			It("should not try to update or create new conditions", func() {
				initialConditionName := alertspolicy.Spec.Conditions[0].Name
				alertspolicy.Spec.Conditions[0].Name = ""
				alertspolicy.Spec.Conditions[0].Namespace = ""

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile twice to be fully settled
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())

				Expect(endStateAlertsPolicy.Status.AppliedSpec.Conditions[0].Name).To(Equal(initialConditionName))
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Status.AppliedSpec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Status.AppliedSpec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Name).To(Equal(initialConditionName))
			})
		})

		Context("and updating that alertspolicy", func() {
			BeforeEach(func() {
				alertspolicy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})

			It("should successfully update", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(1))

			})
		})

		Context("and updating a condition name", func() {
			BeforeEach(func() {
				//clear out the Name and Namespace since those aren't stored in yaml so are blank when applying yaml
				alertspolicy.Spec.Conditions[0].Name = ""
				alertspolicy.Spec.Conditions[0].Namespace = ""
				alertspolicy.Spec.Conditions[0].Spec.Name = "New conditionName"
			})

			It("should create a new condition with the new name", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("New conditionName"))
				Expect(endStateCondition.Name).ToNot(Equal(originalConditionName))
			})

			It("should delete the old condition ", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())

				var originalCondition nrv1.AlertsNrqlCondition
				originalConditionNamespaceType := types.NamespacedName{
					Name:      originalConditionName,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, originalConditionNamespaceType, &originalCondition)
				Expect(err).ToNot(BeNil())
				Expect(originalCondition.Spec.Name).To(Equal(""))
			})
		})

		Context("and updating a condition ", func() {
			BeforeEach(func() {
				//clear out the Name and Namespace since those aren't stored in yaml so are blank when appling yaml
				alertspolicy.Spec.Conditions[0].Name = ""
				alertspolicy.Spec.Conditions[0].Namespace = ""
				alertspolicy.Spec.Conditions[0].Spec.Nrql.Query = "SELECT count(*) FROM MyEvent"
			})

			It("should update the existing AlertsNrqlCondition with the updated spec", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Nrql.Query).To(Equal("SELECT count(*) FROM MyEvent"))
				Expect(endStateCondition.Name).To(Equal(originalConditionName))
			})

			It("should set the inherited values on the updated condition", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
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
				secondConditionSpec := nrv1.AlertsPolicyConditionSpec{
					AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
						Terms: []nrv1.AlertsNrqlConditionTerm{
							{
								ThresholdDuration:    30,
								Operator:             "above",
								Priority:             "critical",
								Threshold:            "5",
								ThresholdOccurrences: alerts.ThresholdOccurrences.All,
							},
						},
						Type:       "NRQL",
						Name:       "second alert condition",
						RunbookURL: "http://test.com/runbook",
						Enabled:    true,
					},
					AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{
						Nrql: alerts.NrqlConditionQuery{
							Query:            "SELECT 1 FROM MyEvents",
							EvaluationOffset: 5,
						},
						ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
						ViolationTimeLimit: "60",
						ExpectedGroups:     2,
						IgnoreOverlap:      true,
					},
					AlertsAPMSpecificSpec:      nrv1.AlertsAPMSpecificSpec{},
					AlertsBaselineSpecificSpec: nrv1.AlertsBaselineSpecificSpec{},
				}
				secondCondition := nrv1.AlertsPolicyCondition{
					Spec: secondConditionSpec,
				}

				alertspolicy.Spec.Conditions = append(alertspolicy.Spec.Conditions, secondCondition)
			})

			It("should add second condition ", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy //test-alertspolicy1942898816
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy) //1942898816
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[1].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[1].Namespace,
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
				mockAlertsClient.UpdatePolicyMutationStub = func(int, string, alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("oh no")
				}
				alertspolicy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})

			It("should return an error", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).To(HaveOccurred())
			})
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When starting with an existing alertspolicy with an APM condition", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.AlertsPolicyConditionSpec{
				AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
					Terms: []nrv1.AlertsNrqlConditionTerm{
						{
							ThresholdDuration:    30,
							Operator:             "above",
							Priority:             "critical",
							Threshold:            "1.5",
							ThresholdOccurrences: alerts.ThresholdOccurrences.All,
						},
					},
					Type:       "apm_app_metric",
					Name:       "APM Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{},
				AlertsAPMSpecificSpec: nrv1.AlertsAPMSpecificSpec{
					Metric:              "Custom/foo",
					UserDefined:         alerts.ConditionUserDefined{},
					Scope:               "application",
					Entities:            []string{"333"},
					GCMetric:            "",
					ViolationCloseTimer: 60,
				},
			}

			alertspolicy = &nrv1.AlertsPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-alertspolicy",
					Namespace: "default",
				},
				Spec: nrv1.AlertsPolicySpec{
					Name:               "test alertspolicy",
					APIKey:             "112233",
					IncidentPreference: "PER_POLICY",
					Region:             "us",
					Conditions: []nrv1.AlertsPolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.AlertsPolicyStatus{
					AppliedSpec: &nrv1.AlertsPolicySpec{},
					PolicyID:    "",
				},
			}

			err := k8sClient.Create(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockAlertsClient.CreatePolicyMutationCallCount()).To(Equal(1))
			Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, alertspolicy)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and making no changes ", func() {
			It("should not try to update or create new conditions", func() {
				initialConditionName := alertspolicy.Spec.Conditions[0].Name
				alertspolicy.Spec.Conditions[0].Name = ""
				alertspolicy.Spec.Conditions[0].Namespace = ""

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile twice to be fully settled
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsAPMCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())

				Expect(endStateAlertsPolicy.Status.AppliedSpec.Conditions[0].Name).To(Equal(initialConditionName))
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Status.AppliedSpec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Status.AppliedSpec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Name).To(Equal(initialConditionName))
				Expect(endStateCondition.Spec.Name).To(Equal("APM Condition"))
			})
		})

		Context("and updating that alertspolicy", func() {
			BeforeEach(func() {
				alertspolicy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})

			It("should successfully update", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(1))
			})
		})

		Context("and updating a condition name", func() {
			BeforeEach(func() {
				//clear out the Name and Namespace since those aren't stored in yaml so are blank when applying yaml
				alertspolicy.Spec.Conditions[0].Name = ""
				alertspolicy.Spec.Conditions[0].Namespace = ""
				alertspolicy.Spec.Conditions[0].Spec.Name = "New conditionName"
			})

			It("should create a new apm condition with the new name", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsAPMCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("New conditionName"))
				Expect(endStateCondition.Name).ToNot(Equal(originalConditionName))
			})

			It("should delete the old condition ", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())

				var originalCondition nrv1.AlertsAPMCondition
				originalConditionNamespaceType := types.NamespacedName{
					Name:      originalConditionName,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, originalConditionNamespaceType, &originalCondition)
				Expect(err).ToNot(BeNil())
				Expect(originalCondition.Spec.Name).To(Equal(""))
			})
		})

		Context("and updating a condition ", func() {
			BeforeEach(func() {
				//clear out the Name and Namespace since those aren't stored in yaml so are blank when appling yaml
				alertspolicy.Spec.Conditions[0].Name = ""
				alertspolicy.Spec.Conditions[0].Namespace = ""
				alertspolicy.Spec.Conditions[0].Spec.Metric = "Custom/bar"
			})

			It("should update the existing AlertsAPMCondition with the updated spec", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsAPMCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Metric).To(Equal("Custom/bar"))
				Expect(endStateCondition.Name).To(Equal(originalConditionName))
			})

			It("should set the inherited values on the updated condition", func() {
				originalConditionName := alertspolicy.Status.AppliedSpec.Conditions[0].Name

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsAPMCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
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
				secondConditionSpec := nrv1.AlertsPolicyConditionSpec{
					AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
						Terms: []nrv1.AlertsNrqlConditionTerm{
							{
								ThresholdDuration:    30,
								Operator:             "above",
								Priority:             "critical",
								Threshold:            "1.5",
								ThresholdOccurrences: alerts.ThresholdOccurrences.All,
							},
						},
						Type:       "apm_app_metric",
						Name:       "Second APM Condition",
						RunbookURL: "http://test.com/runbook",
						Enabled:    true,
					},
					AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{},
					AlertsAPMSpecificSpec: nrv1.AlertsAPMSpecificSpec{
						Metric:              "Custom/foo",
						UserDefined:         alerts.ConditionUserDefined{},
						Scope:               "application",
						Entities:            []string{"333"},
						GCMetric:            "",
						ViolationCloseTimer: 60,
					},
				}
				secondCondition := nrv1.AlertsPolicyCondition{
					Spec: secondConditionSpec,
				}

				alertspolicy.Spec.Conditions = append(alertspolicy.Spec.Conditions, secondCondition)
			})

			It("should add second condition ", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy //test-alertspolicy1942898816
				var endStateCondition nrv1.AlertsAPMCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy) //1942898816
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[1].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[1].Namespace,
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
				mockAlertsClient.UpdatePolicyMutationStub = func(int, string, alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("oh no")
				}
				alertspolicy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})

			It("should return an error", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).To(HaveOccurred())
			})
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When starting with an existing alertspolicy with two NRQL conditions", func() {
		BeforeEach(func() {
			conditionSpec = &nrv1.AlertsPolicyConditionSpec{
				AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
					Terms: []nrv1.AlertsNrqlConditionTerm{
						{
							ThresholdDuration:    30,
							Operator:             "above",
							Priority:             "critical",
							Threshold:            "5",
							ThresholdOccurrences: alerts.ThresholdOccurrences.All,
						},
					},
					Type:       "NRQL",
					Name:       "NRQL Condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{
					Nrql: alerts.NrqlConditionQuery{
						Query:            "SELECT 1 FROM MyEvents",
						EvaluationOffset: 5,
					},
					ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
					ViolationTimeLimit: "60",
					ExpectedGroups:     2,
					IgnoreOverlap:      true,
				},
				AlertsAPMSpecificSpec:      nrv1.AlertsAPMSpecificSpec{},
				AlertsBaselineSpecificSpec: nrv1.AlertsBaselineSpecificSpec{},
			}

			alertspolicy = &nrv1.AlertsPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-alertspolicy",
					Namespace: "default",
				},
				Spec: nrv1.AlertsPolicySpec{
					Name:               "test alertspolicy",
					APIKey:             "112233",
					IncidentPreference: "PER_POLICY",
					Region:             "us",
					Conditions: []nrv1.AlertsPolicyCondition{
						{
							Spec: *conditionSpec,
						},
					},
				},
				Status: nrv1.AlertsPolicyStatus{
					AppliedSpec: &nrv1.AlertsPolicySpec{},
					PolicyID:    "",
				},
			}

			secondConditionSpec := nrv1.AlertsPolicyConditionSpec{
				AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
					Terms: []nrv1.AlertsNrqlConditionTerm{
						{
							ThresholdDuration:    30,
							Operator:             "above",
							Priority:             "critical",
							Threshold:            "5",
							ThresholdOccurrences: alerts.ThresholdOccurrences.All,
						},
					},
					Type:       "NRQL",
					Name:       "second alert condition",
					RunbookURL: "http://test.com/runbook",
					Enabled:    true,
				},
				AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{
					Nrql: alerts.NrqlConditionQuery{
						Query:            "SELECT 1 FROM MyEvents",
						EvaluationOffset: 5,
					},
					ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
					ViolationTimeLimit: "60",
					ExpectedGroups:     2,
					IgnoreOverlap:      true,
				},
				AlertsAPMSpecificSpec:      nrv1.AlertsAPMSpecificSpec{},
				AlertsBaselineSpecificSpec: nrv1.AlertsBaselineSpecificSpec{},
			}
			secondCondition := nrv1.AlertsPolicyCondition{
				Spec: secondConditionSpec,
			}

			alertspolicy.Spec.Conditions = append(alertspolicy.Spec.Conditions, secondCondition)

			err := k8sClient.Create(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())
			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockAlertsClient.CreatePolicyMutationCallCount()).To(Equal(1))
			Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))

			// change the event after creation via reconciliation
			err = k8sClient.Get(ctx, namespacedName, alertspolicy)
			Expect(err).ToNot(HaveOccurred())

			deletedConditionNamespace = types.NamespacedName{
				Namespace: alertspolicy.Spec.Conditions[1].Namespace,
				Name:      alertspolicy.Spec.Conditions[1].Name,
			}
		})

		Context("and removing the second condition ", func() {
			BeforeEach(func() {
				alertspolicy.Spec.Conditions = []nrv1.AlertsPolicyCondition{alertspolicy.Spec.Conditions[0]}
			})

			It("should remove second condition ", func() {
				alertspolicy.Spec.Conditions[0].Name = ""
				alertspolicy.Spec.Conditions[0].Namespace = ""

				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy)
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("NRQL Condition"))
				Expect(len(endStateAlertsPolicy.Spec.Conditions)).To(Equal(1))
				var deletedCondition nrv1.AlertsNrqlCondition
				Expect(deletedConditionNamespace.Name).ToNot(Equal(endStateCondition.Name))
				err = k8sClient.Get(ctx, deletedConditionNamespace, &deletedCondition)
				Expect(err).ToNot(BeNil())
				Expect(deletedCondition.Name).To(Equal(""))
			})

			It("should not call the alerts API ", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockAlertsClient.UpdatePolicyMutationCallCount()).To(Equal(0))
			})
		})

		Context("and removing the first condition ", func() {
			BeforeEach(func() {
				alertspolicy.Spec.Conditions = []nrv1.AlertsPolicyCondition{alertspolicy.Spec.Conditions[1]}

			})

			It("should remove first condition ", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				var endStateAlertsPolicy nrv1.AlertsPolicy //test-alertspolicy1942898816
				var endStateCondition nrv1.AlertsNrqlCondition
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsPolicy) //1942898816
				Expect(err).To(BeNil())
				conditionNameType := types.NamespacedName{
					Name:      endStateAlertsPolicy.Spec.Conditions[0].Name,
					Namespace: endStateAlertsPolicy.Spec.Conditions[0].Namespace,
				}
				err = k8sClient.Get(ctx, conditionNameType, &endStateCondition)
				Expect(err).To(BeNil())
				Expect(endStateCondition.Spec.Name).To(Equal("second alert condition"))
			})
		})

		Context("and when the alerts client returns an error", func() {
			BeforeEach(func() {
				mockAlertsClient.UpdatePolicyMutationStub = func(int, string, alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error) {
					return &alerts.AlertsPolicy{}, errors.New("oh no")
				}
				alertspolicy.Spec.IncidentPreference = "PER_CONDITION_AND_TARGET"
			})

			It("should return an error", func() {
				err := k8sClient.Update(ctx, alertspolicy)
				Expect(err).ToNot(HaveOccurred())

				// Need to call reconcile to update the condition
				_, err = r.Reconcile(request)
				Expect(err).To(HaveOccurred())
			})
		})

		AfterEach(func() {
			err := k8sClient.Delete(ctx, alertspolicy)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("diffIntSlice", func() {
		var (
			in      []int
			compare []int
			diff    []int
		)

		Context("with same slices", func() {
			BeforeEach(func() {
				in = []int{1, 2}
				compare = []int{1, 2}
			})

			It("returns a zero slice", func() {
				diff = diffIntSlice(in, compare)
				Expect(diff).To(Equal([]int{}))
			})
		})

		Context("with 2 same and 1 different value", func() {
			BeforeEach(func() {
				in = []int{1, 2, 3}
				compare = []int{1, 2}
			})

			It("returns expected difference", func() {
				diff = diffIntSlice(in, compare)
				Expect(diff).To(Equal([]int{3}))
			})
		})

		Context("with all different values", func() {
			BeforeEach(func() {
				in = []int{1, 2}
				compare = []int{3, 4}
			})

			It("returns expected difference", func() {
				diff = diffIntSlice(in, compare)
				Expect(diff).To(Equal([]int{1, 2}))
			})
		})
	})
})
