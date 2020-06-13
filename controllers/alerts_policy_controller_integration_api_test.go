// +build integration

package controllers

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/stretchr/testify/require"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	"github.com/newrelic/newrelic-kubernetes-operator/internal/testutil"
)

func TestIntegrationAlertsPolicyController(t *testing.T) {
	t.Parallel()

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
}
