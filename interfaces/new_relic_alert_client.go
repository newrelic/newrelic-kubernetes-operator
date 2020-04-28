package interfaces

import (
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/region"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . NewRelicAlertsClient
type NewRelicAlertsClient interface {
	CreateNrqlCondition(int, alerts.NrqlCondition) (*alerts.NrqlCondition, error)
	UpdateNrqlCondition(alerts.NrqlCondition) (*alerts.NrqlCondition, error)
	ListNrqlConditions(int) ([]*alerts.NrqlCondition, error)
	DeleteNrqlCondition(int) (*alerts.NrqlCondition, error)
	GetPolicy(id int) (*alerts.Policy, error)
	CreatePolicy(alerts.Policy) (*alerts.Policy, error)
	UpdatePolicy(alerts.Policy) (*alerts.Policy, error)
	DeletePolicy(int) (*alerts.Policy, error)
	ListPolicies(*alerts.ListPoliciesParams) ([]alerts.Policy, error)
}

func InitializeAlertsClient(apiKey string, regionName string) (NewRelicAlertsClient, error) {
	configuration := config.New()
	configuration.AdminAPIKey = apiKey

	regName, err := region.Parse(regionName)
	if err != nil {
		return nil, err
	}
	reg, err := region.Get(regName)
	if err != nil {
		return nil, err
	}
	err = configuration.SetRegion(reg)
	if err != nil {
		return nil, err
	}

	alertsClientthing := alerts.New(configuration)
	return &alertsClientthing, nil
}

//PartialAPIKey - Returns a partial API key to ensure we don't log the full API Key
func PartialAPIKey(apiKey string) string {
	partialKeyLength := min(10, len(apiKey))
	return apiKey[0:partialKeyLength] + "..."
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
