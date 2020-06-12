package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("AlertsChannel reconciliation", func() {
	BeforeEach(func() {
		err := ignoreAlreadyExists(k8sClient.Create(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		}))
		Expect(err).ToNot(HaveOccurred())
	})

	var (
		ctx            context.Context
		r              *AlertsChannelReconciler
		alertsChannel  *nrv1.AlertsChannel
		request        ctrl.Request
		namespacedName types.NamespacedName
		// secret         *v1.Secret
		fakeAlertFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
	)
	BeforeEach(func() {
		ctx = context.Background()

		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}

		alertsClient.CreateChannelStub = func(a alerts.Channel) (*alerts.Channel, error) {
			a.ID = 543
			return &a, nil
		}
		alertsClient.ListChannelsStub = func() ([]*alerts.Channel, error) {
			return []*alerts.Channel{}, nil
		}

		alertsClient.ListPoliciesStub = func(*alerts.ListPoliciesParams) ([]alerts.Policy, error) {
			return []alerts.Policy{
				{
					ID:   1122,
					Name: "my-policy-name",
				},
			}, nil
		}

		fakeAlertFunc = func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}

		r = &AlertsChannelReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
		}

		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "myalertschannel",
		}

		alertsChannel = &nrv1.AlertsChannel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myalertschannel",
				Namespace: "default",
			},
			Spec: nrv1.AlertsChannelSpec{
				Name:         "my alert channel",
				APIKey:       "api-key",
				APIKeySecret: nrv1.NewRelicAPIKeySecret{},
				Region:       "US",
				Type:         "email",
				Links: nrv1.ChannelLinks{
					PolicyIDs: []int{
						1,
						2,
					},
					PolicyNames: []string{
						"my-policy-name",
					},
					PolicyKubernetesObjects: []metav1.ObjectMeta{
						{
							Name:      "my-policy",
							Namespace: "default",
						},
					},
				},
				Configuration: nrv1.AlertsChannelConfiguration{
					Recipients: "me@email.com",
				},
			},

			Status: nrv1.AlertsChannelStatus{
				AppliedSpec: &nrv1.AlertsChannelSpec{},
				ChannelID:   0,
			},
		}
		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "myalertschannel",
		}
		request = ctrl.Request{NamespacedName: namespacedName}

		testPolicy := nrv1.AlertsPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-policy",
				Namespace: "default",
			},
			Status: nrv1.AlertsPolicyStatus{
				AppliedSpec: &nrv1.AlertsPolicySpec{},
				PolicyID:    665544,
			},
		}
		err := ignoreAlreadyExists(k8sClient.Create(ctx, &testPolicy))
		Expect(err).ToNot(HaveOccurred())

	})

	Context("When starting with no alertsChannels", func() {

		Context("and given a new alertsChannel", func() {
			Context("with a valid alertsChannelSpec", func() {
				It("should create that alertsChannel via the AlertClient", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(alertsClient.CreateChannelCallCount()).To(Equal(1))
				})

				It("updates the ChannelID on the kubernetes object", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateAlertsChannel nrv1.AlertsChannel
					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).To(BeNil())
					Expect(endStateAlertsChannel.Status.ChannelID).To(Equal(543))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					var endStateAlertsChannel nrv1.AlertsChannel
					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).To(BeNil())
					Expect(endStateAlertsChannel.Status.AppliedSpec).To(Equal(&alertsChannel.Spec))
				})

				It("adds a policy by policy name to the alertsChannel", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					alertsChannelAPI := alertsClient.CreateChannelArgsForCall(0)
					Expect(alertsChannelAPI.Links.PolicyIDs).To(ContainElement(1122))

				})

				It("adds a policy by k8s object reference to the alertsChannel", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					alertsChannelAPI := alertsClient.CreateChannelArgsForCall(0)
					Expect(alertsChannelAPI.Links.PolicyIDs).To(ContainElement(665544))
				})
			})

		})

		Context("and given as new alertsChannel that exists in New Relic", func() {
			BeforeEach(func() {
				alertsClient.ListChannelsStub = func() ([]*alerts.Channel, error) {
					return []*alerts.Channel{
						{
							ID:   112233,
							Name: "my alert channel",
						},
					}, nil
				}

			})
			It("Should not create a new AlertsChannel in New Relic", func() {

				err := k8sClient.Create(ctx, alertsChannel)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				Expect(alertsClient.ListChannelsCallCount()).To(Equal(1))
				Expect(alertsClient.CreateChannelCallCount()).To(Equal(0))

			})
			It("Should update the ChannelId on the kubernetes object", func() {

				err := k8sClient.Create(ctx, alertsChannel)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				var endStateAlertsChannel nrv1.AlertsChannel
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
				Expect(err).To(BeNil())
				Expect(endStateAlertsChannel.Status.ChannelID).To(Equal(112233))
			})
			It("Should update the AppliedSpec on the kubernetes object", func() {

				err := k8sClient.Create(ctx, alertsChannel)
				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
				var endStateAlertsChannel nrv1.AlertsChannel
				err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
				Expect(err).To(BeNil())
				Expect(endStateAlertsChannel.Status.AppliedSpec).To(Equal(&alertsChannel.Spec))
			})
		})

		AfterEach(func() {
			// Delete the alertsChannel
			err := k8sClient.Delete(ctx, alertsChannel)
			Expect(err).ToNot(HaveOccurred())

			// Need to call reconcile to delete finalizer
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("When starting with an existinng alertsChannel", func() {
		BeforeEach(func() {
			err := k8sClient.Create(ctx, alertsChannel)
			Expect(err).ToNot(HaveOccurred())

			// call reconcile
			_, err = r.Reconcile(request)
			Expect(err).ToNot(HaveOccurred())

		})

		Context("and deleting that alertsChannel", func() {
			BeforeEach(func() {
				err := k8sClient.Delete(ctx, alertsChannel)

				Expect(err).ToNot(HaveOccurred())

				// call reconcile
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should delete via the NR API", func() {
				Expect(alertsClient.DeleteChannelCallCount()).To(Equal(1))
			})

			It("Should delete the k8s object", func() {
				var endStateAlertsChannel nrv1.AlertsChannel
				err := k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(" \"myalertschannel\" not found"))
				Expect(endStateAlertsChannel.Name).To(Equal(""))
			})
		})

		Context("and updating that alertsChannel", func() {
			BeforeEach(func() {
				//Get the object again after creation
				err := k8sClient.Get(ctx, namespacedName, alertsChannel)
				Expect(err).ToNot(HaveOccurred())
			})

			PContext("When adding a new policyID to the list of policyIds", func() {

				BeforeEach(func() {
					alertsChannel.Spec.Links.PolicyIDs = append(alertsChannel.Spec.Links.PolicyIDs, 4)
					err := k8sClient.Update(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should call the NR API", func() {
					Expect(alertsClient.UpdatePolicyChannelsCallCount()).To(Equal(1))
				})
				It("Should update the appliedSpec", func() {
					var endStateAlertsChannel nrv1.AlertsChannel
					err := k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).ToNot(HaveOccurred())
					Expect(endStateAlertsChannel.Status.AppliedSpec).To(Equal(alertsChannel.Spec))
				})
			})

			AfterEach(func() {
				err := k8sClient.Delete(ctx, alertsChannel)
				Expect(err).ToNot(HaveOccurred())
				_, err = r.Reconcile(request)
				Expect(err).ToNot(HaveOccurred())
			})

		})

	})

	// 	Context("and given a new NrqlAlertCondition", func() {
	// 		Context("with a valid alertsChannel and a kubernetes secret", func() {
	// 			BeforeEach(func() {
	// 				alertsChannel.Spec.APIKey = ""
	// 				alertsChannel.Spec.APIKeySecret = nrv1.NewRelicAPIKeySecret{
	// 					Name:      "my-api-key-secret",
	// 					Namespace: "my-namespace",
	// 					KeyName:   "my-api-key",
	// 				}

	// 				secret = &v1.Secret{
	// 					ObjectMeta: metav1.ObjectMeta{
	// 						Name:      "my-api-key-secret",
	// 						Namespace: "my-namespace",
	// 					},
	// 					Data: map[string][]byte{
	// 						"my-api-key": []byte("data_here"),
	// 					},
	// 				}
	// 				Expect(ignoreAlreadyExists(k8sClient.Create(ctx, secret))).To(Succeed())
	// 			})
	// 			It("should create that alertsChannel via the AlertClient", func() {

	// 				err := k8sClient.Create(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
	// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
	// 			})
	// 			AfterEach(func() {
	// 				//k8sClient.Delete(ctx, secret)

	// 			})

	// 			It("updates the ConditionID on the kubernetes object", func() {
	// 				err := k8sClient.Create(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(BeNil())
	// 				Expect(endStateCondition.Status.ConditionID).To(Equal(111))
	// 			})

	// 			It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
	// 				err := k8sClient.Create(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(BeNil())
	// 				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&alertsChannel.Spec))
	// 			})
	// 		})
	// 	})

	// 	Context("and given a NrqlAlertCondition that exists in New Relic", func() {
	// 		JustBeforeEach(func() {
	// 			alertsChannel = &nrv1.NrqlAlertCondition{
	// 				ObjectMeta: metav1.ObjectMeta{
	// 					Name:      "test-alertsChannel",
	// 					Namespace: "default",
	// 				},
	// 				Spec: nrv1.NrqlAlertConditionSpec{
	// 					GenericConditionSpec: nrv1.GenericConditionSpec{
	// 						Terms: []nrv1.AlertConditionTerm{
	// 							{
	// 								Duration:     "30",
	// 								Operator:     "above",
	// 								Priority:     "critical",
	// 								Threshold:    "5",
	// 								TimeFunction: "all",
	// 							},
	// 						},
	// 						Type:       "NRQL",
	// 						Name:       "NRQL Condition matches",
	// 						RunbookURL: "http://test.com/runbook",
	// 						ID:         777,

	// 						Enabled:          true,
	// 						ExistingPolicyID: 42,
	// 						APIKey:           "112233",
	// 					},
	// 					NrqlSpecificSpec: nrv1.NrqlSpecificSpec{
	// 						ViolationCloseTimer: 60,
	// 						ExpectedGroups:      2,
	// 						IgnoreOverlap:       true,
	// 						ValueFunction:       "max",
	// 						Nrql: nrv1.NrqlQuery{
	// 							Query:      "SELECT 1 FROM MyEvents",
	// 							SinceValue: "5",
	// 						},
	// 					},
	// 				},
	// 				Status: nrv1.NrqlAlertConditionStatus{
	// 					AppliedSpec: &nrv1.NrqlAlertConditionSpec{},
	// 					ConditionID: 0,
	// 				},
	// 			}

	// 		})

	// 		Context("with a valid alertsChannel", func() {

	// 			It("does not create a new alertsChannel", func() {
	// 				err := k8sClient.Create(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(0))
	// 			})

	// 			It("updates the ConditionID on the kubernetes object", func() {
	// 				err := k8sClient.Create(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(BeNil())
	// 				Expect(endStateCondition.Status.ConditionID).To(Equal(112))
	// 			})

	// 			It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
	// 				err := k8sClient.Create(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(BeNil())
	// 				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&alertsChannel.Spec))
	// 			})

	// 		})
	// 	})

	// 	Context("and alertsChannel has already been created", func() {
	// 		BeforeEach(func() {
	// 			err := k8sClient.Create(ctx, alertsChannel)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			// call reconcile
	// 			_, err = r.Reconcile(request)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
	// 			Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))

	// 			// change the event after creation via reconciliation
	// 			err = k8sClient.Get(ctx, namespacedName, alertsChannel)
	// 			Expect(err).ToNot(HaveOccurred())
	// 		})

	// 		Context("when alertsChannel has been changed", func() {
	// 			BeforeEach(func() {
	// 				alertsChannel.Spec.ID = 0
	// 				alertsChannel.Spec.Nrql.Query = "SELECT 1 FROM NewEventType"
	// 			})

	// 			It("updates the alertsChannel via the client", func() {
	// 				err := k8sClient.Update(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// Only call count for Update is changed from second reconciliation run
	// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
	// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(1))
	// 			})

	// 			It("allows the API to change the ConditionID on the kubernetes object", func() {
	// 				err := k8sClient.Update(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(BeNil())
	// 				Expect(endStateCondition.Status.ConditionID).To(Equal(112))
	// 			})

	// 			It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
	// 				err := k8sClient.Update(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(BeNil())
	// 				Expect(endStateCondition.Status.AppliedSpec).To(Equal(&alertsChannel.Spec))
	// 			})
	// 		})

	// 		Context("when alertsChannel has not changed", func() {
	// 			It("does not make an API call with the client", func() {
	// 				err := k8sClient.Update(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1))
	// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
	// 			})
	// 		})
	// 	})

	// 	AfterEach(func() {
	// 		// Delete the alertsChannel
	// 		err := k8sClient.Delete(ctx, alertsChannel)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		// Need to call reconcile to delete finalizer
	// 		_, err = r.Reconcile(request)
	// 		Expect(err).ToNot(HaveOccurred())
	// 	})
	// })

	// Context("When starting with an existing alertsChannel", func() {

	// 	Context("and deleting a NrqlAlertCondition", func() {
	// 		BeforeEach(func() {
	// 			err := k8sClient.Create(ctx, alertsChannel)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			// call reconcile
	// 			_, err = r.Reconcile(request)
	// 			Expect(err).ToNot(HaveOccurred())

	// 			// change the event after creation via reconciliation
	// 			err = k8sClient.Get(ctx, namespacedName, alertsChannel)
	// 			Expect(err).ToNot(HaveOccurred())

	// 		})
	// 		Context("with a valid alertsChannel", func() {
	// 			It("should delete that alertsChannel via the AlertClient", func() {
	// 				err := k8sClient.Delete(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
	// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
	// 				Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))
	// 			})

	// 			It("should return an error when getting the object", func() {
	// 				err := k8sClient.Delete(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(HaveOccurred())
	// 			})

	// 		})
	// 		Context("with a alertsChannel with no alertsChannel ID", func() {

	// 			BeforeEach(func() {
	// 				alertsChannel.Status.ConditionID = 0
	// 				err := k8sClient.Update(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 			})
	// 			It("should just remove the finalizer and delete", func() {
	// 				err := k8sClient.Delete(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
	// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
	// 				Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(0))

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(HaveOccurred())
	// 				Expect(endStateCondition.Name).To(Equal(""))
	// 			})

	// 		})

	// 		Context("when the Alerts API reports no alertsChannel found ", func() {

	// 			BeforeEach(func() {
	// 				alertsClient.DeleteNrqlConditionStub = func(int) (*alerts.NrqlCondition, error) {
	// 					return &alerts.NrqlCondition{}, errors.New("resource not found")
	// 				}

	// 			})
	// 			It("should just remove the finalizer and delete", func() {
	// 				err := k8sClient.Delete(ctx, alertsChannel)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				// call reconcile
	// 				_, err = r.Reconcile(request)
	// 				Expect(err).ToNot(HaveOccurred())

	// 				Expect(alertsClient.CreateNrqlConditionCallCount()).To(Equal(1)) //This is 1 because the create occurring in the
	// 				Expect(alertsClient.UpdateNrqlConditionCallCount()).To(Equal(0))
	// 				Expect(alertsClient.DeleteNrqlConditionCallCount()).To(Equal(1))

	// 				var endStateCondition nrv1.NrqlAlertCondition
	// 				err = k8sClient.Get(ctx, namespacedName, &endStateCondition)
	// 				Expect(err).To(HaveOccurred())
	// 				Expect(endStateCondition.Name).To(Equal(""))
	// 			})

	// 		})

	// 	})

})
