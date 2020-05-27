package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/testhelpers"
	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func AlertsPolicyTestSetup(t *testing.T) client.Client {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "configs", "crd", "bases")},
	}

	// var err error
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	err = nrv1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	// +kubebuilder:scaffold:scheme

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, k8sClient)

	return k8sClient
}

func NewTestAlertsPolicy(t *testing.T) *nrv1.AlertsPolicy {
	envAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")
	envAccountID := os.Getenv("NEW_RELIC_ACCOUNT_ID")

	if envAPIKey == "" || envAccountID == "" {
		t.Skipf("acceptance testing requires NEW_RELIC_API_KEY and NEW_RELIC_ACCOUNT_ID")
	}

	accountID, err := strconv.Atoi(envAccountID)
	assert.NoError(t, err)

	if envRegion == "" {
		envRegion = "us"
	}

	policyName := fmt.Sprintf("test-policy-%s", testhelpers.RandSeq(5))

	return &nrv1.AlertsPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: "default",
		},
		Spec: nrv1.AlertsPolicySpec{
			APIKey:             envAPIKey,
			AccountID:          accountID,
			IncidentPreference: "PER_POLICY",
			Name:               policyName,
			Region:             envRegion,
			// Conditions: []nrv1.AlertsPolicyCondition{
			// 	{
			// 		Spec: *conditionSpec,
			// 	},
			// },
		},
		Status: nrv1.AlertsPolicyStatus{
			AppliedSpec: &nrv1.AlertsPolicySpec{},
			// PolicyID:    0,
		},
	}

}

func NewTestAlertsNrqlCondition(t *testing.T) *nrv1.AlertsNrqlCondition {
	envAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")
	envAccountID := os.Getenv("NEW_RELIC_ACCOUNT_ID")

	if envAPIKey == "" || envAccountID == "" {
		t.Skipf("acceptance testing requires NEW_RELIC_API_KEY and NEW_RELIC_ACCOUNT_ID")
	}

	accountID, err := strconv.Atoi(envAccountID)
	assert.NoError(t, err)

	if envRegion == "" {
		envRegion = "us"
	}

	conditionName := fmt.Sprintf("test-condition-%s", testhelpers.RandSeq(5))

	return &nrv1.AlertsNrqlCondition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      conditionName,
			Namespace: "default",
		},

		// TODO make use of NewTestAlertsNrqlConditionSpec()
		Spec: nrv1.AlertsNrqlConditionSpec{
			APIKey:    envAPIKey,
			AccountID: accountID,
			Terms: []nrv1.AlertsNrqlConditionTerm{
				{
					Operator:             alerts.NrqlConditionOperators.Above,
					Priority:             alerts.NrqlConditionPriorities.Critical,
					Threshold:            "5",
					ThresholdDuration:    60,
					ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
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
			ViolationTimeLimit: alerts.NrqlConditionViolationTimeLimits.OneHour,
			Region:             envRegion,
			ExpectedGroups:     2,
			IgnoreOverlap:      true,
			Enabled:            true,
		},
		Status: nrv1.AlertsNrqlConditionStatus{
			AppliedSpec: &nrv1.AlertsNrqlConditionSpec{},
			// ConditionID: 0,
		},
	}

}

func NewTestAlertsNrqlConditionSpec(t *testing.T) *nrv1.AlertsNrqlConditionSpec {
	return &nrv1.AlertsNrqlConditionSpec{
		Terms: []nrv1.AlertsNrqlConditionTerm{
			{
				Operator:             alerts.NrqlConditionOperators.Above,
				Priority:             alerts.NrqlConditionPriorities.Critical,
				Threshold:            "5",
				ThresholdDuration:    60,
				ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
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
		ViolationTimeLimit: alerts.NrqlConditionViolationTimeLimits.OneHour,
		ExpectedGroups:     2,
		IgnoreOverlap:      true,
		Enabled:            true,
		// ExistingPolicyID:   integrationPolicy.ID,
		// APIKey:             integrationAlertsConfig.PersonalAPIKey,
		// AccountID:          accountID,
	}
}
