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

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("AlertsChannel reconciliation", func() {
	var (
		ctx            context.Context
		r              *AlertsChannelReconciler
		alertsChannel  *nrv1.AlertsChannel
		request        ctrl.Request
		namespacedName types.NamespacedName
		err            error
		// secret         *v1.Secret
		fakeAlertFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
		testPolicy    nrv1.AlertsPolicy
	)

	BeforeEach(func() {
		err = ignoreAlreadyExists(k8sClient.Create(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-namespace",
			},
		}))
		Expect(err).ToNot(HaveOccurred())

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
		newRelicAgent := newrelic.Application{}

		r = &AlertsChannelReconciler{
			Client:          k8sClient,
			Log:             logf.Log,
			AlertClientFunc: fakeAlertFunc,
			NewRelicAgent:   newRelicAgent,
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
				AppliedSpec:      &nrv1.AlertsChannelSpec{},
				ChannelID:        0,
				AppliedPolicyIDs: []int{},
			},
		}
		namespacedName = types.NamespacedName{
			Namespace: "default",
			Name:      "myalertschannel",
		}
		request = ctrl.Request{NamespacedName: namespacedName}

		testPolicy = nrv1.AlertsPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-policy",
				Namespace: "default",
			},
			Status: nrv1.AlertsPolicyStatus{
				AppliedSpec: &nrv1.AlertsPolicySpec{},
				PolicyID:    "665544",
			},
		}
		err = ignoreAlreadyExists(k8sClient.Create(ctx, &testPolicy))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("When starting with no alertsChannels", func() {
		Context("and given a new alertsChannel", func() {
			Context("with a valid email alertsChannelSpec", func() {
				BeforeEach(func() {
					err = k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
				})

				It("should create that alertsChannel via the AlertClient", func() {
					Expect(alertsClient.CreateChannelCallCount()).To(Equal(1))
				})

				It("updates the ChannelID on the kubernetes object", func() {
					var endStateAlertsChannel nrv1.AlertsChannel
					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).To(BeNil())
					Expect(endStateAlertsChannel.Status.ChannelID).To(Equal(543))
				})

				It("updates the AppliedSpec on the kubernetes object for later comparison", func() {
					var endStateAlertsChannel nrv1.AlertsChannel
					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).To(BeNil())
					Expect(endStateAlertsChannel.Status.AppliedSpec).To(Equal(&alertsChannel.Spec))
				})

				It("adds a policy by policy name to the alertsChannel", func() {
					var endStateAlertsChannel nrv1.AlertsChannel
					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).To(BeNil())
					Expect(endStateAlertsChannel.Status.AppliedPolicyIDs).To(ContainElement(1122))
				})

				It("adds a policy by k8s object reference to the alertsChannel", func() {
					var endStateAlertsChannel nrv1.AlertsChannel
					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).To(BeNil())
					Expect(endStateAlertsChannel.Status.AppliedPolicyIDs).To(ContainElement(665544))
				})
			})

			Context("with a valid webhook alertsChannelSpec", func() {
				var endStateAlertsChannel nrv1.AlertsChannel
				BeforeEach(func() {
					alertsChannel.Spec.Type = "webhook"
					alertsChannel.Spec.Configuration.Headers = []nrv1.ChannelHeader{
						{Name: "HEADER", Value: "value"},
					}
					alertsChannel.Spec.Configuration.Payload = map[string]string{
						"details": "$EVENT_DETAILS",
					}

					err = k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())

					err = k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).To(BeNil())
				})

				It("sets the channel type to webhook", func() {
					Expect(endStateAlertsChannel.Spec.Type).To(Equal("webhook"))
				})

				It("sets the channel header configuration", func() {
					header := endStateAlertsChannel.Spec.Configuration.Headers[0]
					Expect(header.Name).To(Equal("HEADER"))
					Expect(header.Value).To(Equal("value"))
				})

				It("sets the channel payload configuration", func() {
					payload := endStateAlertsChannel.Spec.Configuration.Payload
					Expect(payload["details"]).To(Equal("$EVENT_DETAILS"))
				})
			})

			Context("and given the same policy via policyID and policy name", func() {
				BeforeEach(func() {
					alertsChannel.Spec.Links = nrv1.ChannelLinks{
						PolicyIDs: []int{
							1122,
						},
						PolicyNames: []string{
							"my-policy-name",
						},
					}
				})

				It("should only create a single link object", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(alertsClient.UpdatePolicyChannelsCallCount()).To(Equal(1))
				})
			})

			Context("and given a AlertsChannel with k8s policy reference that has no policyID", func() {
				var existingPolicyID string

				BeforeEach(func() {
					key := types.NamespacedName{Name: "my-policy",
						Namespace: "default"}
					err := k8sClient.Get(ctx, key, &testPolicy)
					Expect(err).ToNot(HaveOccurred())
					existingPolicyID = testPolicy.Status.PolicyID
					testPolicy.Status.PolicyID = ""
					err = k8sClient.Update(ctx, &testPolicy)
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should fail the reconcile loop", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())
					_, err = r.Reconcile(request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Retrieved policy " + testPolicy.Name + " but ID was blank"))
				})

				AfterEach(func() {
					key := types.NamespacedName{Name: "my-policy",
						Namespace: "default"}
					err := k8sClient.Get(ctx, key, &testPolicy)
					Expect(err).ToNot(HaveOccurred())
					testPolicy.Status.PolicyID = existingPolicyID
					err = k8sClient.Update(ctx, &testPolicy)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("and given as new alertsChannel that exists in New Relic", func() {
			Context("when the existing Channel is the same as the configuration", func() {
				BeforeEach(func() {
					alertsClient.ListChannelsStub = func() ([]*alerts.Channel, error) {
						return []*alerts.Channel{
							{
								ID:   112233,
								Name: "my alert channel",
								Type: "email",
								Configuration: alerts.ChannelConfiguration{
									Recipients: "me@email.com",
								},
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

			Context("when the existing Channel is different from the configuration", func() {
				BeforeEach(func() {
					alertsClient.ListChannelsStub = func() ([]*alerts.Channel, error) {
						return []*alerts.Channel{
							{
								ID:   112233,
								Name: "my alert channel",
								Type: "email",
								Configuration: alerts.ChannelConfiguration{
									Recipients: "me@stuff.com",
								},
							},
						}, nil
					}
					alertsClient.CreateChannelStub = func(a alerts.Channel) (*alerts.Channel, error) {
						a.ID = 112244
						return &a, nil
					}

				})

				It("Should delete and create a new AlertsChannel in New Relic", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(alertsClient.ListChannelsCallCount()).To(Equal(1))
					Expect(alertsClient.CreateChannelCallCount()).To(Equal(1))
					Expect(alertsClient.DeleteChannelCallCount()).To(Equal(1))
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
					Expect(endStateAlertsChannel.Status.ChannelID).To(Equal(112244))
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

			Context("when multiple existing Channels are returned from the alerts API", func() {
				BeforeEach(func() {
					alertsClient.ListChannelsStub = func() ([]*alerts.Channel, error) {
						return []*alerts.Channel{
							{
								ID:   112233,
								Name: "my alert channel",
								Type: "email",
								Configuration: alerts.ChannelConfiguration{
									Recipients: "me@stuff.com",
								},
							},
							{
								ID:   112245,
								Name: "my alert channel",
								Type: "email",
								Configuration: alerts.ChannelConfiguration{
									Recipients: "me2@stuff.com",
								},
							},
						}, nil
					}
					alertsClient.CreateChannelStub = func(a alerts.Channel) (*alerts.Channel, error) {
						a.ID = 112234
						return &a, nil
					}
				})

				It("Should delete both and create a new AlertsChannel in New Relic", func() {
					err := k8sClient.Create(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())

					// call reconcile
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
					Expect(alertsClient.ListChannelsCallCount()).To(Equal(1))
					Expect(alertsClient.CreateChannelCallCount()).To(Equal(1))
					Expect(alertsClient.DeleteChannelCallCount()).To(Equal(2))
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
					Expect(endStateAlertsChannel.Status.ChannelID).To(Equal(112234))
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

	Context("When starting with an existing alertsChannel", func() {
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

			Context("When adding a new policyID to the list of policyIds", func() {
				BeforeEach(func() {
					alertsChannel.Spec.Links.PolicyIDs = append(alertsChannel.Spec.Links.PolicyIDs, 4)
					err := k8sClient.Update(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should call the NR API", func() {
					Expect(alertsClient.UpdatePolicyChannelsCallCount()).To(Equal(5)) //4 existing plus 1 new policy
					policyID, _ := alertsClient.UpdatePolicyChannelsArgsForCall(4)
					Expect(policyID).To(Equal(4))
				})

				It("Should update the appliedPolicyIDs", func() {
					var endStateAlertsChannel nrv1.AlertsChannel
					err := k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).ToNot(HaveOccurred())
					Expect(endStateAlertsChannel.Status.AppliedPolicyIDs).To(ContainElement(4))
				})
			})

			Context("When removing a policyID to the list of policyIds", func() {
				BeforeEach(func() {
					alertsChannel.Spec.Links.PolicyIDs = []int{1}
					err := k8sClient.Update(ctx, alertsChannel)
					Expect(err).ToNot(HaveOccurred())
					_, err = r.Reconcile(request)
					Expect(err).ToNot(HaveOccurred())
				})

				It("Should call the NR API", func() {
					Expect(alertsClient.DeletePolicyChannelCallCount()).To(Equal(1))
				})

				It("Should update the appliedPolicyIDs", func() {
					var endStateAlertsChannel nrv1.AlertsChannel
					err := k8sClient.Get(ctx, namespacedName, &endStateAlertsChannel)
					Expect(err).ToNot(HaveOccurred())
					Expect(endStateAlertsChannel.Status.AppliedPolicyIDs).ToNot(ContainElement(2))
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
})
