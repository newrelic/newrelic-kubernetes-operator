package interfaces

import (
	"fmt"

	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/region"

	"github.com/newrelic/newrelic-kubernetes-operator/internal/info"
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

	// NerdGraph
	CreatePolicyMutation(accountID int, policy alerts.AlertsPolicyInput) (*alerts.AlertsPolicy, error)
	UpdatePolicyMutation(accountID int, policyID int, policy alerts.AlertsPolicyUpdateInput) (*alerts.AlertsPolicy, error)
	DeletePolicyMutation(accountID, id int) (*alerts.AlertsPolicy, error)
	QueryPolicySearch(accountID int, params alerts.AlertsPoliciesSearchCriteriaInput) ([]*alerts.AlertsPolicy, error)

	CreateNrqlConditionStaticMutation(accountID int, policyID int, nrqlCondition alerts.NrqlConditionInput) (*alerts.NrqlAlertCondition, error)
	UpdateNrqlConditionStaticMutation(accountID int, conditionID int, nrqlCondition alerts.NrqlConditionInput) (*alerts.NrqlAlertCondition, error)
	DeleteConditionMutation(accountID int, conditionID int) (string, error)
	SearchNrqlConditionsQuery(accountID int, searchCriteria alerts.NrqlConditionsSearchCriteria) ([]*alerts.NrqlAlertCondition, error)
}

func NewClient(apiKey string, regionValue string) (*newrelic.NewRelic, error) {
	cfg := config.New()

	client, err := newrelic.New(
		newrelic.ConfigPersonalAPIKey(apiKey),
		newrelic.ConfigLogLevel(cfg.LogLevel),
		newrelic.ConfigRegion(region.Name(regionValue)),
		newrelic.ConfigUserAgent(info.UserAgent()),
		newrelic.ConfigServiceName(info.Name),
	)

	if err != nil {
		return nil, err
	}

	return client, nil
}

func InitializeAlertsClient(apiKey string, regionName string) (NewRelicAlertsClient, error) {
	client, err := NewClient(apiKey, regionName)
	if err != nil {
		return nil, fmt.Errorf("unable to create New Relic client with error: %s", err)
	}

	return &client.Alerts, nil
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
