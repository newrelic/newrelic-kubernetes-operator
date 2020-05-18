// +build integration

package v1

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/region"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var alertsClient *interfacesfakes.FakeNewRelicAlertsClient
var integrationAlertsClient alerts.Alerts
var integrationAlertsConfig config.Config
var integrationPolicy *alerts.Policy
var randomString string

var _ = Describe("NrqlAlertConditionSpec", func() {
	var condition NrqlAlertConditionSpec

	BeforeEach(func() {

		integrationAlertsConfig = NewIntegrationTestConfig()
		integrationAlertsClient = alerts.New(integrationAlertsConfig)
		accountID, err := strconv.Atoi(os.Getenv("NEW_RELIC_ACCOUNT_ID"))
		Expect(err).ToNot(HaveOccurred())

		randomString = RandSeq(5)
		// Create the policy that we will work with.
		alertPolicy := alerts.Policy{
			Name:               fmt.Sprintf("k8s-test-integration-nrql-policy-ztest-%s", randomString),
			IncidentPreference: "PER_POLICY",
		}
		integrationPolicy, err := integrationAlertsClient.CreatePolicy(alertPolicy)
		Expect(err).To(BeNil())

		condition = NrqlAlertConditionSpec{
			Terms: []AlertConditionTerm{
				{
					// Duration:          resource.MustParse("30"),
					Operator:          "above",
					Priority:          "critical",
					Threshold:         "5",
					ThresholdDuration: 30,
					TimeFunction:      "all",
				},
			},
			Nrql: alerts.NrqlConditionQuery{
				Query:            "SELECT 1 FROM MyEvents",
				EvaluationOffset: 5,
			},
			Type:               "NRQL",
			Name:               "NRQL Condition",
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
		}
	})

	Describe("APINrqlAlertCondition", func() {
		It("converts NrqlAlertConditionSpec object to NrqlCondition object from go client, retaining field values", func() {
			apiCondition := condition.APINrqlAlertCondition()

			Expect(fmt.Sprint(reflect.TypeOf(apiCondition))).To(Equal("alerts.NrqlAlertCondition"))

			Expect(apiCondition.Type).To(Equal("alerts.NrqlConditionType"))
			Expect(apiCondition.Name).To(Equal("NRQL Condition"))
			Expect(apiCondition.RunbookURL).To(Equal("http://test.com/runbook"))
			Expect(apiCondition.ValueFunction).To(Equal(alerts.NrqlConditionValueFunctions.SingleValue))
			// Expect(apiCondition.ID).To(Equal(777))
			Expect(apiCondition.ViolationTimeLimit).To(Equal(alerts.NrqlConditionViolationTimeLimits.OneHour))
			// Expect(apiCondition.ExpectedGroups).To(Equal(2))
			// Expect(apiCondition.IgnoreOverlap).To(Equal(true))
			Expect(apiCondition.Enabled).To(Equal(true))

			apiTerm := apiCondition.Terms[0]

			Expect(fmt.Sprint(reflect.TypeOf(apiTerm))).To(Equal("alerts.ConditionTerm"))

			// Expect(apiTerm.Duration).To(Equal(30))
			Expect(apiTerm.Operator).To(Equal(alerts.OperatorTypes.Above))
			Expect(apiTerm.Priority).To(Equal(alerts.PriorityTypes.Critical))
			Expect(apiTerm.Threshold).To(Equal(float64(5)))
			// Expect(apiTerm.TimeFunction).To(Equal(alerts.TimeFunctionTypes.All))

			// apiQuery := apiCondition.Nrql
			//
			// Expect(fmt.Sprint(reflect.TypeOf(apiQuery))).To(Equal("alerts.NrqlQuery"))
			//
			// Expect(apiQuery.Query).To(Equal("SELECT 1 FROM MyEvents"))
			// Expect(apiQuery.SinceValue).To(Equal("5"))
		})
	})
})

const (
	LogLevel  = "debug"                                                     // LogLevel used in mock configs
	UserAgent = "newrelic/newrelic-kubernetes-operator (automated testing)" // UserAgent used in mock configs
)

func NewIntegrationTestConfig() config.Config {
	envPersonalAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envAdminAPIKey := os.Getenv("NEW_RELIC_ADMIN_API_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")

	if envPersonalAPIKey == "" && envAdminAPIKey == "" {
		log.Error(
			fmt.Errorf("acceptance testing requires NEW_RELIC_API_KEY and NEW_RELIC_ADMIN_API_KEY"),
			"incomplete environment",
			"NEW_RELIC_ADMIN_API_KEY", envAdminAPIKey,
			"NEW_RELIC_API_KEY", envPersonalAPIKey,
		)
	}

	cfg := config.New()

	// Set some defaults
	cfg.LogLevel = LogLevel
	cfg.UserAgent = UserAgent

	cfg.PersonalAPIKey = envPersonalAPIKey
	cfg.AdminAPIKey = envAdminAPIKey

	if envRegion != "" {
		regName, err := region.Parse(envRegion)

		reg, err := region.Get(regName)
		if err != nil {
			log.Error(err, "getting region", "region", reg)
		}

		err = cfg.SetRegion(reg)
		if err != nil {
			log.Error(err, "setting region", "region", reg)
		}
	}

	return cfg
}

func RandSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)

}
