package v1

import (
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("AlertsChannel_webhook", func() {
	var (
		r            AlertsChannel
		alertsClient *interfacesfakes.FakeNewRelicAlertsClient
	)

	BeforeEach(func() {
		k8Client = testk8sClient
		alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}
		fakeAlertFunc := func(string, string) (interfaces.NewRelicAlertsClient, error) {
			return alertsClient, nil
		}
		alertClientFunc = fakeAlertFunc
		r = AlertsChannel{
			ObjectMeta: v1.ObjectMeta{
				Name: "test alert channel",
			},
			Spec: AlertsChannelSpec{
				ID:           88,
				Name:         "my alert channel",
				APIKey:       "api-key",
				APIKeySecret: NewRelicAPIKeySecret{},
				Region:       "US",
				Type:         "email",
				Links: ChannelLinks{
					PolicyIDs: []int{
						1,
						2,
					},
				},
				Configuration: AlertsChannelConfiguration{
					Recipients: "me@email.com",
				},
			},
		}

		alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
			return &alerts.Policy{
				ID: 46286,
			}, nil
		}
	})
	Context("ValidateCreate", func() {
		Context("With a valid Alert Channel", func() {

			It("Should create the Alert Channel", func() {
				err := r.ValidateCreate()
				Expect(err).ToNot(HaveOccurred())

			})
		})

		Context("With an invalid Region", func() {

			BeforeEach(func() {
				r.Spec.Region = "hamburgers"
			})

			It("Should reject the Alert Channel creation", func() {
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("hamburgers"))
			})
		})

		Context("With an invalid Type", func() {

			BeforeEach(func() {
				r.Spec.Type = "burritos"
			})

			It("Should reject the Alert Channel creation", func() {
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("burritos"))
			})
		})

		Context("With no API Key or secret", func() {

			BeforeEach(func() {
				r.Spec.APIKey = ""
			})

			It("Should reject the Alert Channel creation", func() {
				err := r.ValidateCreate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("either api_key or api_key_secret must be set"))
			})
		})

	})

})
