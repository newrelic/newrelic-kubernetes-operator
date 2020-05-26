// +build integration

package controllers

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/stretchr/testify/require"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

func TestIntegrationAlertsPolicyController(t *testing.T) {
	t.Parallel()

	conditionSpec := &nrv1.AlertsNrqlConditionSpec{
		Terms: []nrv1.AlertsNrqlConditionTerm{
			{
				//Duration:     resource.MustParse("30"),
				Operator:     "above",
				Priority:     "critical",
				Threshold:    resource.MustParse("5"),
				TimeFunction: "all",
			},
		},
		Nrql: nrv1.AlertsNrqlConditionQuery{
			Query:      "SELECT 1 FROM MyEvents",
			SinceValue: "5",
		},
		Type:                "NRQL",
		Name:                "NRQL Condition",
		RunbookURL:          "http://test.com/runbook",
		ValueFunction:       "max",
		ViolationCloseTimer: 60,
		ExpectedGroups:      2,
		IgnoreOverlap:       true,
		Enabled:             true,
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

	// Must come before calling reconciler.Reconcile()
	k8sClient := alertsPolicyTestSetup(t, policy)

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
