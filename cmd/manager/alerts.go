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
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	nrv1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1"
	"github.com/newrelic/newrelic-kubernetes-operator/controllers"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
	// +kubebuilder:scaffold:imports
)

func registerAlerts(mgr *ctrl.Manager) error {
	nrqlReconciler := &controllers.NrqlAlertConditionReconciler{
		Client:          (*mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("NrqlAlertCondition"),
		Scheme:          (*mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := nrqlReconciler.SetupWithManager(*mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NrqlAlertCondition")
		os.Exit(1)
	}

	nrqlAlertCondition := &nrv1.NrqlAlertCondition{}
	if err := nrqlAlertCondition.SetupWebhookWithManager(*mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "NrqlAlertCondition")
		os.Exit(1)
	}

	apmReconciler := &controllers.ApmAlertConditionReconciler{
		Client: (*mgr).GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ApmAlertCondition"),
		Scheme: (*mgr).GetScheme(),
		//AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := apmReconciler.SetupWithManager(*mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApmAlertCondition")
		os.Exit(1)
	}
	apmAlertCondition := &nrv1.ApmAlertCondition{}
	if err := apmAlertCondition.SetupWebhookWithManager(*mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ApmAlertCondition")
		os.Exit(1)
	}

	policyReconciler := &controllers.PolicyReconciler{
		Client:          (*mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("Policy"),
		Scheme:          (*mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := policyReconciler.SetupWithManager(*mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Policy")
		os.Exit(1)
	}

	policy := &nrv1.Policy{}

	if err := policy.SetupWebhookWithManager(*mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Policy")
		os.Exit(1)
	}

	alertsPolicyReconciler := &controllers.AlertsPolicyReconciler{
		Client:          (*mgr).GetClient(),
		Log:             ctrl.Log.WithName("controllers").WithName("AlertsPolicy"),
		Scheme:          (*mgr).GetScheme(),
		AlertClientFunc: interfaces.InitializeAlertsClient,
	}

	if err := alertsPolicyReconciler.SetupWithManager(*mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AlertsPolicy")
		os.Exit(1)
	}

	return nil
}
