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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nralertsv1beta1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1beta1"
)

// NrqlAlertConditionReconciler reconciles a NrqlAlertCondition object
type NrqlAlertConditionReconciler struct {
	client.Client
	Alerts alerts.Alerts
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nr-alerts.k8s.newrelic.com,resources=nrqlalertconditions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nr-alerts.k8s.newrelic.com,resources=nrqlalertconditions/status,verbs=get;update;patch

func (r *NrqlAlertConditionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("nrqlalertcondition", req.NamespacedName)

	r.Log.Info("Starting reconcile action")
	var condition nralertsv1beta1.NrqlAlertCondition
	err := r.Client.Get(ctx, req.NamespacedName, &condition)
	if err != nil {
		r.Log.Error(err, "tried getting condition", "name", req.NamespacedName)
	}

	if condition.Status.AppliedSpec == &condition.Spec {
		return ctrl.Result{}, nil
	}

	r.Log.Info("Reconciling", "condition", condition)

	APICondition := condition.Spec.APICondition()
	r.Log.Info("Trying to create or update condition", "API fields", APICondition)

	if condition.Status.AppliedSpec != nil && &condition.Spec != condition.Status.AppliedSpec {
		APICondition.ID = condition.Status.ConditionID
		_, err := r.Alerts.UpdateNrqlCondition(APICondition)
		if err != nil {
			r.Log.Error(err, "failed to update condition")
		}

		condition.Status.AppliedSpec = &condition.Spec

		err = r.Client.Update(ctx, &condition)
		if err != nil {
			r.Log.Error(err, "tried updating condition status", "name", req.NamespacedName)
		}
	} else {
		createdCondition, err := r.Alerts.CreateNrqlCondition(APICondition)
		if err != nil {
			r.Log.Error(err, "failed to create condition")
		} else {
			condition.Status.AppliedSpec = &condition.Spec
		}

		condition.Status.ConditionID = createdCondition.ID

		err = r.Client.Update(ctx, &condition)
		if err != nil {
			r.Log.Error(err, "tried updating condition status", "name", req.NamespacedName)
		}
	}

	return ctrl.Result{}, nil
}

func (r *NrqlAlertConditionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nralertsv1beta1.NrqlAlertCondition{}).
		Complete(r)
}
