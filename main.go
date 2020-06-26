/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/newrelic/newrelic-kubernetes-operator/controllers"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/internal/info"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = nrv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var showVersion bool
	var devMode bool

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&showVersion, "version", false, "Show version information.")
	flag.BoolVar(&devMode, "dev-mode", false, "Enable development level logging (stacktraces on warnings, no sampling)")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s version %s\n", info.Name, info.Version)
		os.Exit(0)
	}

	logger := zap.New(zap.UseDevMode(devMode))
	ctrl.SetLogger(logger)

	opts := ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "newrelic-kubernetes-operator",
		Port:               9443,
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	// Register Alerts
	//err = registerAlerts(&mgr)
	//if err != nil {
	//	setupLog.Error(err, "unable to register alerts")
	//	os.Exit(1)
	//}

	nrqlAlertConditionReconciler := &controllers.NrqlAlertConditionReconciler{
		Client:          (mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("NrqlAlertCondition"),
		Scheme:          (mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := nrqlAlertConditionReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NrqlAlertCondition")
		os.Exit(1)
	}

	nrqlAlertCondition := &nrv1.NrqlAlertCondition{}
	if err := nrqlAlertCondition.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "NrqlAlertCondition")
		os.Exit(1)
	}

	// alertsnrqlcondition
	alertsNrqlConditionReconciler := &controllers.AlertsNrqlConditionReconciler{
		Client:          mgr.GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("AlertsNrqlCondition"),
		Scheme:          (mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := alertsNrqlConditionReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AlertsNrqlCondition")
		os.Exit(1)
	}

	alertsNrqlCondition := &nrv1.AlertsNrqlCondition{}
	if err := alertsNrqlCondition.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AlertsNrqlCondition")
		os.Exit(1)
	}

	// apmalertcondition
	apmReconciler := &controllers.ApmAlertConditionReconciler{
		Client:          (mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("ApmAlertCondition"),
		Scheme:          (mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := apmReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApmAlertCondition")
		os.Exit(1)
	}

	apmAlertCondition := &nrv1.ApmAlertCondition{}
	if err := apmAlertCondition.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ApmAlertCondition")
		os.Exit(1)
	}

	// alertsapmcondition
	alertsAPMReconciler := &controllers.AlertsAPMConditionReconciler{
		Client:          (mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("AlertsAPMCondition"),
		Scheme:          (mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := alertsAPMReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AlertsAPMCondition")
		os.Exit(1)
	}

	alertsAPMCondition := &nrv1.AlertsAPMCondition{}
	if err := alertsAPMCondition.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AlertsAPMCondition")
		os.Exit(1)
	}

	// policy
	policyReconciler := &controllers.PolicyReconciler{
		Client:          (mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("Policy"),
		Scheme:          (mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := policyReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Policy")
		os.Exit(1)
	}

	policy := &nrv1.Policy{}
	if err := policy.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Policy")
		os.Exit(1)
	}

	//alertsChannel
	alertsChannelReconciler := &controllers.AlertsChannelReconciler{
		Client:          (mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("alertsChannel"),
		Scheme:          (mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := alertsChannelReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "alertsChannel")
		os.Exit(1)
	}

	alertsChannel := &nrv1.AlertsChannel{}
	if err := alertsChannel.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AlertsChannel")
		os.Exit(1)
	}

	// alertspolicy
	alertsPolicyReconciler := &controllers.AlertsPolicyReconciler{
		Client:          (mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("AlertsPolicy"),
		Scheme:          (mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}
	if err := alertsPolicyReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AlertsPolicy")
		os.Exit(1)
	}

	alertsPolicy := &nrv1.AlertsPolicy{}
	if err := alertsPolicy.SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AlertsPolicy")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
