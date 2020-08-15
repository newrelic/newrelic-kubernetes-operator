// +build integration

package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func testSetup(t *testing.T, object runtime.Object) client.Client {
	ctx := context.Background()
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
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

	err = k8sClient.Create(ctx, object)
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

	conditionSpec := &nrv1.ConditionSpec{
		GenericConditionSpec: nrv1.GenericConditionSpec{
			Terms: []nrv1.AlertConditionTerm{
				{
					Duration:     "30",
					Operator:     "above",
					Priority:     "critical",
					Threshold:    "5",
					TimeFunction: "all",
				},
			},
			Type:       "NRQL",
			Name:       "NRQL Condition",
			RunbookURL: "http://test.com/runbook",
			Enabled:    true,
		},
		NrqlSpecificSpec: nrv1.NrqlSpecificSpec{
			Nrql: nrv1.NrqlQuery{
				Query:      "SELECT 1 FROM MyEvents",
				SinceValue: "5",
			},
			ValueFunction:       "max",
			ViolationCloseTimer: 60,
			ExpectedGroups:      2,
			IgnoreOverlap:       true,
		},
		APMSpecificSpec: nrv1.APMSpecificSpec{},
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

func TestIntegrationAlertsChannelController(t *testing.T) {
	t.Parallel()

	envAPIKey := os.Getenv("NEW_RELIC_API_KEY")
	envRegion := os.Getenv("NEW_RELIC_REGION")

	if envAPIKey == "" {
		t.Skipf("acceptance testing requires NEW_RELIC_API_KEY")
	}

	if envRegion == "" {
		envRegion = "us"
	}

	alertsChannel := &nrv1.AlertsChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myalertschannel",
			Namespace: "default",
		},
		Spec: nrv1.AlertsChannelSpec{
			Name:   "my alert channel",
			APIKey: envAPIKey,
			Region: envRegion,
			Type:   "email",
			Configuration: nrv1.AlertsChannelConfiguration{
				Recipients: "me@email.com",
			},
		},

		Status: nrv1.AlertsChannelStatus{
			AppliedSpec:      &nrv1.AlertsChannelSpec{},
			ChannelID:        0,
			AppliedPolicyIDs: []int{},
		},
	}

	// Must come before calling reconciler.Reconcile()
	k8sClient := testSetup(t, alertsChannel)

	namespacedName := types.NamespacedName{
		Namespace: "default",
		Name:      "myalertschannel",
	}

	request := ctrl.Request{
		NamespacedName: namespacedName,
	}

	reconciler := &AlertsChannelReconciler{
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
