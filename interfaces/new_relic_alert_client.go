package interfaces

import "github.com/newrelic/newrelic-client-go/pkg/alerts"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . NewRelicAlertsClient
type NewRelicAlertsClient interface {
	CreateNrqlCondition(alerts.NrqlCondition) (*alerts.NrqlCondition, error)
	UpdateNrqlCondition(alerts.NrqlCondition) (*alerts.NrqlCondition, error)
	ListNrqlConditions(int) (*[]alerts.NrqlCondition, error)
}
