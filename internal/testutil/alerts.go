package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"github.com/newrelic/newrelic-client-go/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
)

func AlertsPolicyTestSetup(t *testing.T) client.Client {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
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
	policyName := fmt.Sprintf("test-policy-%s", testhelpers.RandSeq(5))

	accountID, err := strconv.Atoi(os.Getenv("NEW_RELIC_ACCOUNT_ID"))
	assert.NoError(t, err)

	return &nrv1.AlertsPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: "default",
		},
		Spec: nrv1.AlertsPolicySpec{
			APIKey:             "112233",
			AccountID:          accountID,
			IncidentPreference: "PER_POLICY",
			Name:               policyName,
			Region:             "US",
		},
		Status: nrv1.AlertsPolicyStatus{
			AppliedSpec: &nrv1.AlertsPolicySpec{},
		},
	}

}

func NewTestAlertsNrqlCondition(t *testing.T) *nrv1.AlertsNrqlCondition {
	conditionName := fmt.Sprintf("test-condition-%s", testhelpers.RandSeq(5))
	conditionSpec := NewTestAlertsPolicyConditionSpec(t)

	policyCondition := nrv1.AlertsPolicyCondition{
		Spec: *conditionSpec,
	}

	return &nrv1.AlertsNrqlCondition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      conditionName,
			Namespace: "default",
		},

		Spec: policyCondition.ReturnNrqlConditionSpec(),
		Status: nrv1.AlertsNrqlConditionStatus{
			AppliedSpec: &nrv1.AlertsNrqlConditionSpec{},
		},
	}
}

func NewTestAlertsPolicyConditionSpec(t *testing.T) *nrv1.AlertsPolicyConditionSpec {
	accountID, err := strconv.Atoi(os.Getenv("NEW_RELIC_ACCOUNT_ID"))
	assert.NoError(t, err)

	return &nrv1.AlertsPolicyConditionSpec{
		AlertsGenericConditionSpec: nrv1.AlertsGenericConditionSpec{
			APIKey:    "nraa-key",
			AccountID: accountID,
			Terms: []nrv1.AlertsNrqlConditionTerm{
				{
					Operator:             alerts.AlertsNRQLConditionTermsOperatorTypes.ABOVE,
					Priority:             alerts.NrqlConditionPriorities.Critical,
					Threshold:            "5",
					ThresholdDuration:    60,
					ThresholdOccurrences: alerts.ThresholdOccurrences.AtLeastOnce,
				},
			},
			Type:             "NRQL",
			Name:             "NRQL Condition",
			RunbookURL:       "http://test.com/runbook",
			ID:               777,
			Enabled:          true,
			ExistingPolicyID: "42",
		},
		AlertsNrqlSpecificSpec: nrv1.AlertsNrqlSpecificSpec{
			ViolationTimeLimit: "60",
			ExpectedGroups:     2,
			IgnoreOverlap:      true,
			ValueFunction:      &alerts.NrqlConditionValueFunctions.SingleValue,
			Nrql: alerts.NrqlConditionQuery{
				Query:            "SELECT 1 FROM MyEvents",
				EvaluationOffset: 5,
			},
		},
		AlertsAPMSpecificSpec: nrv1.AlertsAPMSpecificSpec{},
	}
}
