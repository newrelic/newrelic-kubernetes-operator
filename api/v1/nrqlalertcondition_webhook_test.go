// +build integration

package v1

import (
	"errors"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces/interfacesfakes"
)

var _ = Describe("Default", func() {
	var (
		r            NrqlAlertCondition
		alertsClient *interfacesfakes.FakeNewRelicAlertsClient
	)

	BeforeEach(func() {

	})

	Describe("CheckExistingPolicyID", func() {
		BeforeEach(func() {
			alertsClient = &interfacesfakes.FakeNewRelicAlertsClient{}
			fakeAlertFunc := func(string, string) (interfaces.NewRelicAlertsClient, error) {
				return alertsClient, nil
			}
			alertClientFunc = fakeAlertFunc

			r = NrqlAlertCondition{
				Spec: NrqlAlertConditionSpec{
					Terms: []AlertConditionTerm{
						{
							Duration:     resource.MustParse("30"),
							Operator:     "above",
							Priority:     "critical",
							Threshold:    resource.MustParse("5"),
							TimeFunction: "all",
						},
					},
					Nrql: NrqlQuery{
						Query:      "SELECT 1 FROM MyEvents",
						SinceValue: "5",
					},
					Type:                "NRQL",
					Name:                "NRQL Condition",
					RunbookURL:          "http://test.com/runbook",
					ValueFunction:       "max",
					ID:                  777,
					ViolationCloseTimer: 60,
					ExpectedGroups:      2,
					IgnoreOverlap:       true,
					Enabled:             true,
					ExistingPolicyID:    42,
				},
			}
		})
		Context("With a valid API Key", func() {
			BeforeEach(func() {
				alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
					return &alerts.Policy{
						ID: 42,
					}, nil
				}
			})
			It("verifies existing policies exist", func() {
				err := r.CheckExistingPolicyID()
				Expect(err).To(BeNil())
				Expect(r.Spec.ExistingPolicyID).To(Equal(42))
			})
		})
		Context("With an invalid API Key", func() {
			BeforeEach(func() {
				alertsClient.GetPolicyStub = func(int) (*alerts.Policy, error) {
					return nil, errors.New("401 response returned: The API key provided is invalid")
				}
			})
			It("returns an error", func() {
				err := r.CheckExistingPolicyID()
				Expect(err).To(Not(BeNil()))
			})
		})
	})
})
