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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"

	customErrors "github.com/newrelic/newrelic-kubernetes-operator/errors"
)

// AlertsPolicyLog is for emitting logs in this package.
var AlertsPolicyLog = logf.Log.WithName("alerts-policy-resource")
var defaultAlertsPolicyIncidentPreference = alerts.AlertsIncidentPreferenceTypes.PER_POLICY

func (r *AlertsPolicy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-nr-k8s-newrelic-com-v1-alertspolicy,mutating=true,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertspolicies,verbs=create;update,versions=v1,name=malertspolicy.kb.io,sideEffects=None

var _ webhook.Defaulter = &AlertsPolicy{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AlertsPolicy) Default() {
	AlertsPolicyLog.Info("default", "name", r.Name)

	if r.Status.AppliedSpec == nil {
		AlertsPolicyLog.Info("Setting null Applied Spec to empty interface")
		r.Status.AppliedSpec = &AlertsPolicySpec{}
	}

	r.DefaultIncidentPreference()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-nr-k8s-newrelic-com-v1-alertspolicy,mutating=false,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertspolicies,versions=v1,name=valertspolicy.kb.io,sideEffects=None

var _ webhook.Validator = &AlertsPolicy{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsPolicy) ValidateCreate() error {
	AlertsPolicyLog.Info("validate create", "name", r.Name)

	collectedErrors := new(customErrors.ErrorCollector)
	err := r.CheckForAPIKeyOrSecret()

	if err != nil {
		collectedErrors.Collect(err)
	}

	err = r.CheckForDuplicateConditions()
	if err != nil {
		collectedErrors.Collect(err)
	}

	err = r.ValidateIncidentPreference()
	if err != nil {
		collectedErrors.Collect(err)
	}

	if len(*collectedErrors) > 0 {
		AlertsPolicyLog.Info("Errors encountered validating policy", "collectedErrors", collectedErrors)
		return collectedErrors
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsPolicy) ValidateUpdate(old runtime.Object) error {
	AlertsPolicyLog.Info("validate update", "name", r.Name)

	collectedErrors := new(customErrors.ErrorCollector)

	err := r.CheckForAPIKeyOrSecret()
	if err != nil {
		collectedErrors.Collect(err)
	}

	err = r.CheckForDuplicateConditions()
	if err != nil {
		collectedErrors.Collect(err)
	}

	err = r.ValidateIncidentPreference()
	if err != nil {
		collectedErrors.Collect(err)
	}

	if len(*collectedErrors) > 0 {
		AlertsPolicyLog.Info("Errors encountered validating policy", "collectedErrors", collectedErrors)
		return collectedErrors
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsPolicy) ValidateDelete() error {
	AlertsPolicyLog.Info("validate delete", "name", r.Name)

	err := r.CheckForAPIKeyOrSecret()
	if err != nil {
		return err
	}

	return nil
}

func (r *AlertsPolicy) DefaultIncidentPreference() {
	if r.Spec.IncidentPreference == "" {
		r.Spec.IncidentPreference = string(defaultAlertsPolicyIncidentPreference)
	}

	r.Spec.IncidentPreference = strings.ToUpper(r.Spec.IncidentPreference)
}

func (r *AlertsPolicy) CheckForDuplicateConditions() error {
	var conditionHashMap = make(map[uint32]bool)

	for _, condition := range r.Spec.Conditions {
		conditionHashMap[condition.SpecHash()] = true
	}

	if len(conditionHashMap) != len(r.Spec.Conditions) {
		AlertsPolicyLog.Info("duplicate conditions detected or hash collision", "conditionHash", conditionHashMap)
		return errors.New("duplicate conditions detected or hash collision")
	}

	return nil
}

func (r *AlertsPolicy) ValidateIncidentPreference() error {
	switch r.Spec.IncidentPreference {
	case "PER_POLICY", "PER_CONDITION", "PER_CONDITION_AND_TARGET":
		return nil
	}

	AlertsPolicyLog.Info("Incident preference must be PER_POLICY, PER_CONDITION, or PER_CONDITION_AND_TARGET", "IncidentPreference value", r.Spec.IncidentPreference)

	return errors.New("incident preference must be PER_POLICY, PER_CONDITION, or PER_CONDITION_AND_TARGET")
}

func (r *AlertsPolicy) CheckForAPIKeyOrSecret() error {
	if r.Spec.APIKey != "" {
		return nil
	}

	if r.Spec.APIKeySecret != (NewRelicAPIKeySecret{}) {
		if r.Spec.APIKeySecret.Name != "" && r.Spec.APIKeySecret.Namespace != "" && r.Spec.APIKeySecret.KeyName != "" {
			return nil
		}
	}

	return errors.New("either api_key or api_key_secret must be set")
}
