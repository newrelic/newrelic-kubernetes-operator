// +build integration

package controllers

import (
	"fmt"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/stretchr/testify/require"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/internal/testutil"
)

func TestIntegrationAlertsPolicyController(t *testing.T) {
	t.Parallel()

	conditionSpec := &nrv1.AlertsNrqlConditionSpec{
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
	}

	policy := &nrv1.AlertsPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
		Spec: nrv1.AlertsPolicySpec{
			Name:               "test policy",
			APIKey:             os.Getenv("NEW_RELIC_API_KEY"),
			IncidentPreference: "PER_POLICY",
			Region:             "us",
			Conditions: []nrv1.AlertsPolicyCondition{
				{
					Spec: *conditionSpec,
				},
			},
		},
		Status: nrv1.AlertsPolicyStatus{
			AppliedSpec: &nrv1.AlertsPolicySpec{},
			PolicyID:    0,
		},
	}

	fmt.Printf("unused policy: %+v", policy)

	// Must come before calling reconciler.Reconcile()
	k8sClient := testutil.AlertsPolicyTestSetup(t)

	namespacedName := types.NamespacedName{
		Namespace: "default",
		Name:      "test-policy",
	}

	request := ctrl.Request{
		NamespacedName: namespacedName,
	}

	reconciler := &AlertsPolicyReconciler{
		Client:          k8sClient,
		Log:             logf.Log,
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	// call reconcile
	_, err := reconciler.Reconcile(request)
	require.NoError(t, err)

	// Deferred teardown
	// defer func() {
	// 	_, err := client.DeletePolicy(policy.ID)

	// 	if err != nil {
	// 		t.Logf("error cleaning up alert policy %d (%s): %s", policy.ID, policy.Name, err)
	// 	}
	// }()
}
