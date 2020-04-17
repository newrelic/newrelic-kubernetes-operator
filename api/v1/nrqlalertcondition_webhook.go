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

package v1

import (
	"errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// log is for logging in this package.
var (
	log             = logf.Log.WithName("nrqlalertcondition-resource")
	alertClientFunc func(string, string) (interfaces.NewRelicAlertsClient, error)
)

func (r *NrqlAlertCondition) SetupWebhookWithManager(mgr ctrl.Manager) error {
	alertClientFunc = interfaces.InitializeAlertsClient

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-nr-alerts-k8s-newrelic-com-v1-nrqlalertcondition,mutating=true,failurePolicy=fail,groups=nr-alerts.k8s.newrelic.com,resources=nrqlalertconditions,verbs=create;update,versions=v1,name=mnrqlalertcondition.kb.io

var _ webhook.Defaulter = &NrqlAlertCondition{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NrqlAlertCondition) Default() {
	log.Info("default", "name", r.Name)

	if r.Status.AppliedSpec == nil {
		log.Info("Setting null Applied Spec to empty interface")
		r.Status.AppliedSpec = &NrqlAlertConditionSpec{}
	}
	log.Info("r.Status.AppliedSpec after", "r.Status.AppliedSpec", r.Status.AppliedSpec)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-nr-alerts-k8s-newrelic-com-v1-nrqlalertcondition,mutating=false,failurePolicy=fail,groups=nr-alerts.k8s.newrelic.com,resources=nrqlalertconditions,versions=v1,name=vnrqlalertcondition.kb.io

var _ webhook.Validator = &NrqlAlertCondition{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NrqlAlertCondition) ValidateCreate() error {
	log.Info("validate create", "name", r.Name)
	return r.CheckExistingPolicyID()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NrqlAlertCondition) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NrqlAlertCondition) ValidateDelete() error {
	log.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *NrqlAlertCondition) CheckExistingPolicyID() error {
	log.Info("Checking existing", "policyId", r.Spec.ExistingPolicyID)

	alertsClient, errAlertClient := alertClientFunc(r.Spec.APIKey, r.Spec.Region)
	if errAlertClient != nil {
		log.Info("failed to get policy", "policyId", r.Spec.ExistingPolicyID, "error", errAlertClient)
		return errAlertClient
	}
	alertPolicy, errAlertPolicy := alertsClient.GetPolicy(r.Spec.ExistingPolicyID)
	if errAlertPolicy != nil {
		log.Info("failed to get policy", "policyId", r.Spec.ExistingPolicyID, "error", errAlertPolicy)
		return errAlertPolicy
	}
	if alertPolicy.ID != r.Spec.ExistingPolicyID {
		return errors.New("Alert policy returned by API did not match")
	}
	return nil
}
