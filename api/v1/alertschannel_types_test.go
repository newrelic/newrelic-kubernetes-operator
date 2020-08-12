package v1

import (
	"context"
	"fmt"
	"reflect"

	"github.com/golang/mock/gomock"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("AlertsChannelSpec", func() {
	Describe("APIChannel", func() {
		var client *interfacesfakes.MockClient

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			client = interfacesfakes.NewMockClient(ctrl)
		})

		Context("email channel", func() {
			var alertsChannelSpec AlertsChannelSpec
			BeforeEach(func() {
				alertsChannelSpec = AlertsChannelSpec{
					ID:     88,
					Name:   "my alert channel",
					APIKey: "api-key",
					Region: "US",
					Type:   "email",
					Links: ChannelLinks{
						PolicyIDs: []int{
							1,
							2,
						},
					},
					Configuration: AlertsChannelConfiguration{
						Recipients: "me@email.com",
					},
				}
			})

			It("converts AlertsChannelSpec object to alerts.Channel object from go client, retaining field values", func() {
				apiChannel, err := alertsChannelSpec.APIChannel(client)
				Expect(err).NotTo(HaveOccurred())

				Expect(fmt.Sprint(reflect.TypeOf(apiChannel))).To(Equal("alerts.Channel"))
				Expect(apiChannel.ID).To(Equal(88))
				Expect(apiChannel.Type).To(Equal(alerts.ChannelTypes.Email))
				Expect(apiChannel.Name).To(Equal("my alert channel"))
				apiConfiguration := apiChannel.Configuration
				Expect(apiConfiguration.Recipients).To(Equal("me@email.com"))
			})
		})

		Context("webhook channel", func() {
			var alertsChannelSpec AlertsChannelSpec
			BeforeEach(func() {
				alertsChannelSpec = AlertsChannelSpec{
					ID:     88,
					Name:   "my alert channel",
					APIKey: "api-key",
					Region: "US",
					Type:   "webhook",
					Links: ChannelLinks{
						PolicyIDs: []int{
							1,
							2,
						},
					},
					Configuration: AlertsChannelConfiguration{
						URL: "https://example.com/",
						Headers: []ChannelHeader{
							{
								Name:  "KEY",
								Value: "VALUE",
							},
							{
								Name:      "SECRET",
								Secret:    "secret",
								Namespace: "default",
								KeyName:   "secret",
							},
						},
					},
				}
			})

			It("converts AlertsChannelSpec object to alerts.Channel object from go client, retaining field values", func() {
				secret := v1.Secret{
					Data: map[string][]byte{
						"secret": []byte("don't tell anyone"),
					},
				}
				key := types.NamespacedName{
					Namespace: "default",
					Name:      "secret",
				}
				client.EXPECT().
					Get(gomock.Eq(context.Background()),
						gomock.Eq(key),
						gomock.AssignableToTypeOf(&secret)).
					SetArg(2, secret)

				apiChannel, err := alertsChannelSpec.APIChannel(client)
				Expect(err).NotTo(HaveOccurred())

				Expect(fmt.Sprint(reflect.TypeOf(apiChannel))).To(Equal("alerts.Channel"))
				Expect(apiChannel.ID).To(Equal(88))
				Expect(apiChannel.Type).To(Equal(alerts.ChannelTypes.Webhook))
				Expect(apiChannel.Name).To(Equal("my alert channel"))
				apiConfiguration := apiChannel.Configuration
				Expect(apiConfiguration.URL).To(Equal("https://example.com/"))
				Expect(apiConfiguration.Headers["KEY"]).To(Equal("VALUE"))
				Expect(apiConfiguration.Headers["SECRET"]).To(Equal("don't tell anyone"))
			})
		})
	})
})
