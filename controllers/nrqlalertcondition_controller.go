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
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nralertsv1beta1 "github.com/newrelic/newrelic-kubernetes-operator/api/v1beta1"
	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// NrqlAlertConditionReconciler reconciles a NrqlAlertCondition object
type NrqlAlertConditionReconciler struct {
	client.Client
	Alerts interfaces.NewRelicAlertsClient
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
		r.Log.Error(err, "tried getting condition", "name", req.NamespacedName.String())
		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		return ctrl.Result{}, nil
	}


	r.Log.Info("Reconciling", "condition", condition)

	//check if condition has condition id
	if condition.Status.ConditionID == 0 {
		r.Log.Info("Checking for existing condition", "conditionName", condition.Name)
		//if no conditionId, get list of conditions and compare name
		existingConditions, err := r.Alerts.ListNrqlConditions(condition.Spec.ExistingPolicyId)
		r.Log.Info("existingConditions", "existingConditions", existingConditions)
		if err != nil {
			r.Log.Error(err, "failed to get list of NRQL conditions from New Relic API")
		} else {
			for _, existingCondition := range *existingConditions {
				if existingCondition.Name == condition.Spec.Name {
					r.Log.Info("Matched on existing condition, updating ConditionId", "conditionId", existingCondition.ID)
					condition.Status.ConditionID = existingCondition.ID
				}
				break
			}
		}

	}

	APICondition := condition.Spec.APICondition()
	r.Log.Info("Trying to create or update condition", "API fields", APICondition)

	if condition.Status.ConditionID != 0 && !reflect.DeepEqual(&condition.Spec, condition.Status.AppliedSpec) {
		r.Log.Info("updating condition")
		APICondition.ID = condition.Status.ConditionID
		updatedCondition, err := r.Alerts.UpdateNrqlCondition(APICondition)
		if err != nil {
			r.Log.Error(err, "failed to update condition")
		} else {
			condition.Status.AppliedSpec = &condition.Spec
			condition.Status.ConditionID = updatedCondition.ID
		}

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
			condition.Status.ConditionID = createdCondition.ID
		}

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
