// +build integration

package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/stretchr/testify/require"

	nralertsv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

func newIntegrationTestClient(t *testing.T) newrelic.NewRelic {
	envAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")

	if envAPIKey == "" {
		t.Skipf("acceptance testing requires NEW_RELIC_API_KEY")
	}

	if envRegion == "" {
		envRegion = "us"
	}

	client, _ := interfaces.NewClient(envAPIKey, envRegion)
	return *client
}

func testSetup(t *testing.T, policy *nrv1.Policy) client.Client {
	ctx := context.Background()
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "configs", "crd", "bases")},
	}

	// var err error
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	err = nralertsv1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	// +kubebuilder:scaffold:scheme

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)
	require.NotNil(t, k8sClient)

	err = k8sClient.Create(ctx, policy)
	require.NoError(t, err)

	return k8sClient
}

func TestIntegrationPolicyController(t *testing.T) {
	t.Parallel()

	envAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")

	if envAPIKey == "" {
		t.Skipf("acceptance testing requires NEW_RELIC_API_KEY")
	}

	if envRegion == "" {
		envRegion = "us"
	}

	conditionSpec := &nrv1.NrqlAlertConditionSpec{
		Terms: []nrv1.AlertConditionTerm{
			{
				Duration:     resource.MustParse("30"),
				Operator:     "above",
				Priority:     "critical",
				Threshold:    resource.MustParse("5"),
				TimeFunction: "all",
			},
		},
		Nrql: nrv1.NrqlQuery{
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

	policy := &nrv1.Policy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
		Spec: nrv1.PolicySpec{
			Name:               "test policy",
			APIKey:             envAPIKey,
			IncidentPreference: "PER_POLICY",
			Region:             envRegion,
			Conditions: []nrv1.PolicyCondition{
				{
					Spec: *conditionSpec,
				},
			},
		},
		Status: nrv1.PolicyStatus{
			AppliedSpec: &nrv1.PolicySpec{},
			PolicyID:    0,
		},
	}

	// Must come before calling reconciler.Reconcile()
	k8sClient := testSetup(t, policy)

	namespacedName := types.NamespacedName{
		Namespace: "default",
		Name:      "test-policy",
	}

	request := ctrl.Request{
		NamespacedName: namespacedName,
	}

	reconciler := &PolicyReconciler{
		Client:          k8sClient,
		Log:             logf.Log,
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	// call reconcile
	result, err := reconciler.Reconcile(request)

	require.NoError(t, err)

	t.Logf("\n\n RESULT: %+v \n\n", result)

	// Deferred teardown
	// defer func() {
	// 	_, err := client.DeletePolicy(policy.ID)

	// 	if err != nil {
	// 		t.Logf("error cleaning up alert policy %d (%s): %s", policy.ID, policy.Name, err)
	// 	}
	// }()
}
