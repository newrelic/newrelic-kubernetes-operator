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

	customErrors "github.com/newrelic/newrelic-kubernetes-operator/errors"
)

// Log is for emitting logs in this package.
var Log = logf.Log.WithName("policy-resource")
var defaultPolicyIncidentPreference = "PER_POLICY"

func (r *Policy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-nr-k8s-newrelic-com-v1-policy,mutating=true,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=policies,verbs=create;update,versions=v1,name=mpolicy.kb.io

var _ webhook.Defaulter = &Policy{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Policy) Default() {
	Log.Info("default", "name", r.Name)

	if r.Status.AppliedSpec == nil {
		log.Info("Setting null Applied Spec to empty interface")
		r.Status.AppliedSpec = &PolicySpec{}
	}

	r.DefaultIncidentPreference()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-nr-k8s-newrelic-com-v1-policy,mutating=false,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=policies,versions=v1,name=vpolicy.kb.io

var _ webhook.Validator = &Policy{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Policy) ValidateCreate() error {
	Log.Info("validate create", "name", r.Name)

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
		Log.Info("Errors encountered validating policy", "collectedErrors", collectedErrors)
		return collectedErrors
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Policy) ValidateUpdate(old runtime.Object) error {
	Log.Info("validate update", "name", r.Name)

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
		Log.Info("Errors encountered validating policy", "collectedErrors", collectedErrors)
		return collectedErrors
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Policy) ValidateDelete() error {
	Log.Info("validate delete", "name", r.Name)

	err := r.CheckForAPIKeyOrSecret()
	if err != nil {
		return err
	}
	return nil
}

func (r *Policy) DefaultIncidentPreference() {
	if r.Spec.IncidentPreference == "" {
		r.Spec.IncidentPreference = defaultPolicyIncidentPreference
	}
	r.Spec.IncidentPreference = strings.ToUpper(r.Spec.IncidentPreference)

}

func (r *Policy) CheckForDuplicateConditions() error {

	var conditionHashMap = make(map[uint32]bool)
	for _, condition := range r.Spec.Conditions {
		conditionHashMap[condition.SpecHash()] = true
	}
	if len(conditionHashMap) != len(r.Spec.Conditions) {
		log.Info("duplicate conditions detected or hash collision", "conditionHash", conditionHashMap)
		return errors.New("duplicate conditions detected or hash collision")
	}

	return nil
}

func (r *Policy) ValidateIncidentPreference() error {
	switch r.Spec.IncidentPreference {
	case "PER_POLICY", "PER_CONDITION", "PER_CONDITION_AND_TARGET":
		return nil
	}
	log.Info("Incident preference must be PER_POLICY, PER_CONDITION, or PER_CONDITION_AND_TARGET", "IncidentPreference value", r.Spec.IncidentPreference)
	return errors.New("incident preference must be PER_POLICY, PER_CONDITION, or PER_CONDITION_AND_TARGET")
}

func (r *Policy) CheckForAPIKeyOrSecret() error {
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
