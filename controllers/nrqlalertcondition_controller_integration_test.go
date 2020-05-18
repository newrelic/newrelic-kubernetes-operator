// +build integration

package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/config"

	nralertsv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var alertsClient *interfacesfakes.FakeNewRelicAlertsClient
var integrationAlertsClient alerts.Alerts
var integrationAlertsConfig config.Config
var integrationPolicy *alerts.Policy
var randomString string

var _ = Describe("NrqlCondition reconciliation", func() {
	var (
		ctx context.Context
		//events         chan string
		r              *NrqlAlertConditionReconciler
		condition      *nralertsv1.NrqlAlertCondition
		request        ctrl.Request
		namespacedName types.NamespacedName
		//expectedEvents []string
		// secret        *v1.Secret
		fakeAlertFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	)

	BeforeEach(func() {
		ctx = context.Background()

		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}
		alertsClient.DeleteNrqlConditionStub = integrationAlertsClient.DeleteNrqlCondition

		alertsClient.SearchNrqlConditionsQueryStub = integrationAlertsClient.SearchNrqlConditionsQuery
		alertsClient.CreateNrqlConditionStaticMutationStub = integrationAlertsClient.CreateNrqlConditionBaselineMutation

		fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return &integrationAlertsClient, nil
		}

		r = &NrqlAlertConditionReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			Alerts:          &integrationAlertsClient,
			AlertClientFunc: fakeAlertFunc,
		}

		integrationAlertsClient = newIntegrationTestClient()
		accountID, err := strconv.Atoi(os.Getenv("NEW_RELIC_ACCOUNT_ID"))
		Expect(err).ToNot(HaveOccurred())

		integrationAlertsConfig = NewIntegrationTestConfig()

		randomString = RandSeq(5)
		// Create the policy that we will work with.
		alertPolicy := alerts.Policy{
			Name:               fmt.Sprintf("k8s-test-integration-nrql-policy-ztest-%s", randomString),
			IncidentPreference: "PER_POLICY",
		}
		integrationPolicy, err := integrationAlertsClient.CreatePolicy(alertPolicy)
		Expect(err).To(BeNil())

		condition = &nralertsv1.NrqlAlertCondition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("k8s-test-condition-%s", randomString),
				Namespace: "default",
			},
			Spec: nralertsv1.NrqlAlertConditionSpec{
				Terms: []nralertsv1.AlertConditionTerm{
					{
						Operator:             alerts.NrqlConditionOperator("ABOVE"),
						Priority:             alerts.NrqlConditionPriority("CRITICAL"),
						Threshold:            "5.1",
						ThresholdDuration:    60,
						ThresholdOccurrences: alerts.ThresholdOccurrence("AT_LEAST_ONCE"),
						TimeFunction:         "all",
					},
				},
				Nrql: alerts.NrqlConditionQuery{
					Query:            "SELECT 1 FROM MyEvents",
					EvaluationOffset: 5,
				},
				// Type:               "NRQL",
				Name:          fmt.Sprintf("NRQL Condition %s", randomString),
				RunbookURL:    "http://test.com/runbook",
				ValueFunction: &alerts.NrqlConditionValueFunctions.SingleValue,
				// ID:                 777,
				ViolationTimeLimit: alerts.NrqlConditionViolationTimeLimits.OneHour,
				ExpectedGroups:     2,
				IgnoreOverlap:      true,
				Enabled:            true,
				ExistingPolicyID:   integrationPolicy.ID,
				APIKey:             integrationAlertsConfig.AdminAPIKey,
				AccountID:          accountID,
			},
			Status: nralertsv1.NrqlAlertConditionStatus{
				AppliedSpec: &nralertsv1.NrqlAlertConditionSpec{},
				ConditionID: 0,
			},
		}

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      fmt.Sprintf("k8s-test-condition-%s", randomString),
		}

		request = ctrl.Request{NamespacedName: namespacedName}
	})

	Context("when starting with no conditions", func() {

		Context("and given a new NrqlAlertCondition", func() {
			Context("with a valid condition", func() {
				It("should make the nerdgraph call to create", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					searchCriteria := alerts.NrqlConditionsSearchCriteria{
						NameLike: condition.Spec.Name,
					}

					// query for the object
					searchResults, err := alertsClient.SearchNrqlConditionsQuery(condition.Spec.AccountID, searchCriteria)
					Expect(err).To(BeNil())
					Expect(len(searchResults)).To(Equal(1))
					Expect(searchResults[0].Name).To(Equal(condition.Spec.Name))
				})

				It("updates the ConditionID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					searchCriteria := alerts.NrqlConditionsSearchCriteria{
						NameLike: condition.Spec.Name,
					}
					searchResults, err := alertsClient.SearchNrqlConditionsQuery(condition.Spec.AccountID, searchCriteria)
					Expect(err).To(BeNil())

					var endStateCondition nralertsv1.NrqlAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(strconv.Itoa(endStateCondition.Status.ConditionID)).To(Equal(searchResults[0].ID))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nralertsv1.NrqlAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a new NrqlAlertCondition", func() {
			Context("with a valid condition and a kubernetes secret", func() {
				It("should make the nerdgraph call to create", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					searchCriteria := alerts.NrqlConditionsSearchCriteria{
						NameLike: condition.Spec.Name,
					}

					// query for the object
					searchResults, err := alertsClient.SearchNrqlConditionsQuery(condition.Spec.AccountID, searchCriteria)
					Expect(err).To(BeNil())
					Expect(len(searchResults)).To(Equal(1))
					Expect(searchResults[0].Name).To(Equal(condition.Spec.Name))
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

					searchCriteria := alerts.NrqlConditionsSearchCriteria{
						NameLike: condition.Spec.Name,
					}
					searchResults, err := alertsClient.SearchNrqlConditionsQuery(condition.Spec.AccountID, searchCriteria)
					Expect(err).To(BeNil())

					var endStateCondition nralertsv1.NrqlAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(strconv.Itoa(endStateCondition.Status.ConditionID)).To(Equal(searchResults[0].ID))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nralertsv1.NrqlAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())
					Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				})
			})
		})

		Context("and given a NrqlAlertCondition that exists in New Relic", func() {
			JustBeforeEach(func() {

				integrationAlertsClient = newIntegrationTestClient()
				accountID, err := strconv.Atoi(os.Getenv("NEW_RELIC_ACCOUNT_ID"))
				Expect(err).ToNot(HaveOccurred())

				integrationAlertsConfig = NewIntegrationTestConfig()

				randomString = RandSeq(5)
				// Create the policy that we will work with.
				alertPolicy := alerts.Policy{
					Name:               fmt.Sprintf("k8s-test-integration-nrql-policy-ztest-%s", randomString),
					IncidentPreference: "PER_POLICY",
				}
				integrationPolicy, err := integrationAlertsClient.CreatePolicy(alertPolicy)
				Expect(err).To(BeNil())

				condition = &nralertsv1.NrqlAlertCondition{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-condition",
						Namespace: "default",
					},
					Spec: nralertsv1.NrqlAlertConditionSpec{
						Terms: []nralertsv1.AlertConditionTerm{
							{
								Operator:             alerts.NrqlConditionOperator("ABOVE"),
								Priority:             alerts.NrqlConditionPriority("CRITICAL"),
								Threshold:            "5.1",
								ThresholdDuration:    60,
								ThresholdOccurrences: alerts.ThresholdOccurrence("AT_LEAST_ONCE"),
								TimeFunction:         "all",
							},
						},
						Nrql: alerts.NrqlConditionQuery{
							Query:            "SELECT 1 FROM MyEvents",
							EvaluationOffset: 5,
						},
						Type:               "NRQL",
						Name:               "NRQL Condition matches",
						RunbookURL:         "http://test.com/runbook",
						ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
						ID:                 777,
						ViolationTimeLimit: alerts.NrqlConditionViolationTimeLimits.OneHour,
						ExpectedGroups:     2,
						IgnoreOverlap:      true,
						Enabled:            true,
						ExistingPolicyID:   integrationPolicy.ID,
						APIKey:             integrationAlertsConfig.PersonalAPIKey,
						AccountID:          accountID,
					},
					Status: nralertsv1.NrqlAlertConditionStatus{
						AppliedSpec: &nralertsv1.NrqlAlertConditionSpec{},
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

					Expect(alertsClient.CreateNrqlConditionStaticMutationCallCount()).To(Equal(0))
				})

				// It("updates the ConditionID on the kubernetes object", func() {
				// 	err := k8sClient.Create(ctx, condition)
				// 	Expect(err).ToNot(HaveOccurred())
				//
				// 	// call reconcile
				// 	_, err = r.Reconcile(request)
				// 	Expect(err).ToNot(HaveOccurred())
				//
				// 	var endStateCondition nralertsv1.NrqlAlertCondition
				// 	err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
				// 	Expect(err).To(BeNil())
				//
				// 	Expect(endStateCondition.Status.ConditionID).To(Equal(112))
				// })

				// It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
				// 	err := k8sClient.Create(ctx, condition)
				// 	Expect(err).ToNot(HaveOccurred())
				//
				// 	// call reconcile
				// 	_, err = r.Reconcile(request)
				// 	Expect(err).ToNot(HaveOccurred())
				//
				// 	var endStateCondition nralertsv1.NrqlAlertCondition
				// 	err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
				// 	Expect(err).To(BeNil())
				// 	Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				// })

			})
		})

		Context("and condition has already been created", func() {
			BeforeEach(func() {
				err := k8sClient.Create(ctx, condition)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())

				// Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
				// Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))

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
					// Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
					// Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(1))
				})

				It("allows the API to change the ConditionID on the kubernetes object", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					searchCriteria := alerts.NrqlConditionsSearchCriteria{
						NameLike: condition.Spec.Name,
					}
					searchResults, err := alertsClient.SearchNrqlConditionsQuery(condition.Spec.AccountID, searchCriteria)
					Expect(err).To(BeNil())

					var endStateCondition nralertsv1.NrqlAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(BeNil())

					Expect(strconv.Itoa(endStateCondition.Status.ConditionID)).To(Equal(searchResults[0].ID))
				})

				// It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
				// 	err := k8sClient.Update(ctx, condition)
				// 	Expect(err).ToNot(HaveOccurred())
				//
				// 	// call reconcile
				// 	_, err = r.Reconcile(request)
				// 	Expect(err).ToNot(HaveOccurred())
				//
				// 	var endStateCondition nralertsv1.NrqlAlertCondition
				// 	err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
				// 	Expect(err).To(BeNil())
				// 	Expect(endStateCondition.Status.AppliedSpec).To(Equal(&condition.Spec))
				// })
			})

			Context("when condition has not changed", func() {
				It("does not make an API call with the client", func() {
					err := k8sClient.Update(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					// Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
					// Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
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

		Context("and deleting a NrqlAlertCondition", func() {
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

					// Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					// Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
					// Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))
				})

				It("should return an error when getting the object", func() {
					err := k8sClient.Delete(ctx, condition)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateCondition nralertsv1.NrqlAlertCondition
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

					// Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					// Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
					// Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(0))

					var endStateCondition nralertsv1.NrqlAlertCondition
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

					// Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
					// Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
					// Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))

					var endStateCondition nralertsv1.NrqlAlertCondition
					err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
					Expect(err).To(HaveOccurred())
					Expect(endStateCondition.Name).To(Equal(""))
				})

			})
		})
	})

})
