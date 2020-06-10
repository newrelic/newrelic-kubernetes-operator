package v1

import (
	"fmt"
	"reflect"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AlertsChannelSpec", func() {
	var alertsChannelSpec AlertsChannelSpec

	BeforeEach(func() {
		alertsChannelSpec = AlertsChannelSpec{
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
		}

	})

	Describe("APICondition", func() {
		It("converts AlertsChannelSpec object to alerts.Channel object from go client, retaining field values", func() {
			apiChannel := alertsChannelSpec.APICondition()

			Expect(fmt.Sprint(reflect.TypeOf(apiChannel))).To(Equal("alerts.Channel"))
			Expect(apiChannel.ID).To(Equal(88))
			Expect(apiChannel.Type).To(Equal(alerts.ChannelTypes.Email))
			Expect(apiChannel.Name).To(Equal("my alert channel"))
			apiLinks := apiChannel.Links
			Expect(apiLinks.PolicyIDs[0]).To(Equal(1))
			Expect(apiLinks.PolicyIDs[1]).To(Equal(2))
			apiConfiguration := apiChannel.Configuration
			Expect(apiConfiguration.Recipients).To(Equal("me@email.com"))
		})
	})
})
