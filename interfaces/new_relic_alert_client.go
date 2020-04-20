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
}

func InitializeAlertsClient(apiKey string, regionName string) (NewRelicAlertsClient, error) {
	configuration := config.Config{
		AdminAPIKey: apiKey,
	}
	err := configuration.SetRegion(region.Parse(regionName))
	if err != nil {
		return nil, err
	}

	alertsClientthing := alerts.New(configuration)
	return &alertsClientthing, nil
}
