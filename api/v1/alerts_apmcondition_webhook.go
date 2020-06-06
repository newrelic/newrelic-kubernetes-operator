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
	"context"
	"errors"
	"strings"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/newrelic/newrelic-kubernetes-operator/interfaces"
)

// log is for logging in this package.
var (
	alertsapmconditionlog = logf.Log.WithName("alertsapmcondition-resource")
)

func (r *AlertsAPMCondition) SetupWebhookWithManager(mgr ctrl.Manager) error {
	alertClientFunc = interfaces.InitializeAlertsClient
	k8Client = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-nr-k8s-newrelic-com-v1-alertsapmcondition,mutating=true,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertsapmconditions,verbs=create;update,versions=v1,name=malertsapmcondition.kb.io,sideEffects=None

var _ webhook.Defaulter = &AlertsAPMCondition{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AlertsAPMCondition) Default() {
	alertsapmconditionlog.Info("default", "name", r.Name)

	if r.Status.AppliedSpec == nil {
		log.Info("Setting null Applied Spec to empty interface")
		r.Status.AppliedSpec = &AlertsAPMConditionSpec{}
	}
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-nr-k8s-newrelic-com-v1-alertsapmcondition,mutating=false,failurePolicy=fail,groups=nr.k8s.newrelic.com,resources=alertsapmconditions,versions=v1,name=valertsapmcondition.kb.io,sideEffects=None

var _ webhook.Validator = &AlertsAPMCondition{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsAPMCondition) ValidateCreate() error {
	alertsapmconditionlog.Info("validate create", "name", r.Name)

	err := r.CheckForAPIKeyOrSecret()
	if err != nil {
		return err
	}

	err = r.CheckRequiredFields()
	if err != nil {
		return err
	}

	var invalidAttributes InvalidAttributeSlice

	invalidAttributes = append(invalidAttributes, r.ValidateType()...)
	invalidAttributes = append(invalidAttributes, r.ValidateMetric()...)
	invalidAttributes = append(invalidAttributes, r.ValidateTerms()...)
	invalidAttributes = append(invalidAttributes, r.ValidateUserDefinedValueFunction()...)

	if len(invalidAttributes) > 0 {
		return errors.New("error with invalid attributes: \n" + invalidAttributes.errorString())
	}
	return r.CheckExistingPolicyID()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsAPMCondition) ValidateUpdate(old runtime.Object) error {
	alertsapmconditionlog.Info("validate update", "name", r)

	err := r.CheckForAPIKeyOrSecret()
	if err != nil {
		return err
	}

	err = r.CheckRequiredFields()
	if err != nil {
		return err
	}
	var invalidAttributes InvalidAttributeSlice

	invalidAttributes = append(invalidAttributes, r.ValidateType()...)
	invalidAttributes = append(invalidAttributes, r.ValidateMetric()...)
	invalidAttributes = append(invalidAttributes, r.ValidateTerms()...)
	invalidAttributes = append(invalidAttributes, r.ValidateUserDefinedValueFunction()...)

	if len(invalidAttributes) > 0 {
		return errors.New("error with invalid attributes")
	}
	return r.CheckExistingPolicyID()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AlertsAPMCondition) ValidateDelete() error {
	alertsapmconditionlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *AlertsAPMCondition) ValidateType() InvalidAttributeSlice {
	switch string(r.Spec.Type) {
	case string(alerts.ConditionTypes.APMApplicationMetric),
		string(alerts.ConditionTypes.APMKeyTransactionMetric),
		string(alerts.ConditionTypes.BrowserMetric),
		string(alerts.ConditionTypes.MobileMetric),
		string(alerts.ConditionTypes.ServersMetric):
		return []invalidAttribute{}
	default:
		log.Info("Invalid Type attribute", "Type", r.Spec.Type)
		return []invalidAttribute{{attribute: "Type", value: string(r.Spec.Type)}}
	}
}

func (r *AlertsAPMCondition) ValidateMetric() InvalidAttributeSlice {
	switch alerts.MetricType(r.Spec.Metric) {
	case alerts.MetricTypes.AjaxResponseTime,
		alerts.MetricTypes.AjaxThroughput,
		alerts.MetricTypes.Apdex,
		alerts.MetricTypes.CPUPercentage,
		alerts.MetricTypes.Database,
		alerts.MetricTypes.DiskIOPercentage,
		alerts.MetricTypes.DomProcessing,
		alerts.MetricTypes.EndUserApdex,
		alerts.MetricTypes.ErrorCount,
		alerts.MetricTypes.ErrorPercentage,
		alerts.MetricTypes.FullestDiskPercentage,
		alerts.MetricTypes.Images,
		alerts.MetricTypes.JSON,
		alerts.MetricTypes.LoadAverageOneMinute,
		alerts.MetricTypes.MemoryPercentage,
		alerts.MetricTypes.MobileCrashRate,
		alerts.MetricTypes.Network,
		alerts.MetricTypes.NetworkErrorPercentage,
		alerts.MetricTypes.PageRendering,
		alerts.MetricTypes.PageViewThroughput,
		alerts.MetricTypes.PageViewsWithJsErrors,
		alerts.MetricTypes.RequestQueuing,
		alerts.MetricTypes.ResponseTime,
		alerts.MetricTypes.ResponseTimeBackground,
		alerts.MetricTypes.ResponseTimeWeb,
		alerts.MetricTypes.StatusErrorPercentage,
		alerts.MetricTypes.Throughput,
		alerts.MetricTypes.ThroughputBackground,
		alerts.MetricTypes.ThroughputWeb,
		alerts.MetricTypes.TotalPageLoad,
		alerts.MetricTypes.UserDefined,
		alerts.MetricTypes.ViewLoading,
		alerts.MetricTypes.WebApplication:
		return []invalidAttribute{}
	default:
		log.Info("Invalid Metric attribute", "Metric", r.Spec.Metric)
		return []invalidAttribute{{attribute: "Type", value: r.Spec.Metric}}
	}
}

func (r *AlertsAPMCondition) ValidateTerms() InvalidAttributeSlice {
	var invalidTerms InvalidAttributeSlice
	for _, term := range r.Spec.APMTerms {
		switch alerts.TimeFunctionType(term.TimeFunction) {
		case alerts.TimeFunctionTypes.All, alerts.TimeFunctionTypes.Any:
			continue
		default:
			log.Info("Invalid UserDefined.ValueFunction passed", "UserDefined.ValueFunction", term.TimeFunction)
			invalidTerms = append(invalidTerms, invalidAttribute{
				attribute: "Term.TimeFunction",
				value:     term.TimeFunction,
			})
		}
		switch alerts.OperatorType(term.Operator) {
		case alerts.OperatorTypes.Equal, alerts.OperatorTypes.Above, alerts.OperatorTypes.Below:
			continue
		default:
			log.Info("Invalid Term.Operator passed", "Term.Operator ", term.Operator)
			invalidTerms = append(invalidTerms, invalidAttribute{
				attribute: "Term.Operator",
				value:     term.Operator,
			})
		}
		switch alerts.PriorityType(term.Priority) {
		case alerts.PriorityTypes.Critical, alerts.PriorityTypes.Warning:
			continue
		default:
			log.Info("Invalid term.Priority passed", "term.Priority", term.Priority)
			invalidTerms = append(invalidTerms, invalidAttribute{
				attribute: "Term.Priority",
				value:     term.Priority,
			})
		}

	}

	return invalidTerms
}

func (r *AlertsAPMCondition) ValidateUserDefinedValueFunction() InvalidAttributeSlice {
	switch r.Spec.UserDefined.ValueFunction {
	case "", alerts.ValueFunctionTypes.Average,
		alerts.ValueFunctionTypes.Max,
		alerts.ValueFunctionTypes.Min,
		alerts.ValueFunctionTypes.SampleSize,
		alerts.ValueFunctionTypes.SingleValue,
		alerts.ValueFunctionTypes.Total:
		return []invalidAttribute{}
	default:
		log.Info("Invalid UserDefined.ValueFunction passed", "UserDefined.ValueFunction", r.Spec.UserDefined.ValueFunction)
		return []invalidAttribute{{attribute: "UserDefined.ValueFunction: ", value: string(r.Spec.UserDefined.ValueFunction)}}
	}
}

func (r *AlertsAPMCondition) CheckExistingPolicyID() error {
	log.Info("Checking existing", "policyId", r.Spec.ExistingPolicyID)
	ctx := context.Background()
	var apiKey string
	if r.Spec.APIKey == "" {
		key := types.NamespacedName{Namespace: r.Spec.APIKeySecret.Namespace, Name: r.Spec.APIKeySecret.Name}
		var apiKeySecret v1.Secret
		getErr := k8Client.Get(ctx, key, &apiKeySecret)
		if getErr != nil {
			log.Error(getErr, "Error getting secret")
			return getErr
		}
		apiKey = string(apiKeySecret.Data[r.Spec.APIKeySecret.KeyName])

	} else {
		apiKey = r.Spec.APIKey
	}

	alertsClient, errAlertClient := alertClientFunc(apiKey, r.Spec.Region)
	if errAlertClient != nil {
		log.Error(errAlertClient, "failed to get policy",
			"policyId", r.Spec.ExistingPolicyID,
			"API Key", interfaces.PartialAPIKey(apiKey),
			"region", r.Spec.Region,
		)
		return errAlertClient
	}
	alertPolicy, errAlertPolicy := alertsClient.QueryPolicy(r.Spec.AccountID, r.Spec.ExistingPolicyID)
	if errAlertPolicy != nil {
		if r.GetDeletionTimestamp() != nil {
			log.Info("Deleting resource", "errAlertPolicy", errAlertPolicy)
			if strings.Contains(errAlertPolicy.Error(), "no alert policy found for id") {
				log.Info("ExistingAlertPolicy not found but we are deleting the condition so this is ok")
				return nil
			}
		}
		log.Error(errAlertPolicy, "failed to get policy",
			"policyId", r.Spec.ExistingPolicyID,
			"API Key", interfaces.PartialAPIKey(apiKey),
			"region", r.Spec.Region,
		)
		return errAlertPolicy
	}

	if alertPolicy.ID != r.Spec.ExistingPolicyID {
		log.Info("Alert policy returned by the API failed to match provided policy ID")
		return errors.New("alert policy returned by API did not match")
	}
	return nil
}

func (r *AlertsAPMCondition) CheckForAPIKeyOrSecret() error {
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

func (r *AlertsAPMCondition) CheckRequiredFields() error {

	missingFields := []string{}
	if r.Spec.Region == "" {
		missingFields = append(missingFields, "region")
	}
	if r.Spec.ExistingPolicyID == "" {
		missingFields = append(missingFields, "existing_policy_id")
	}
	if len(missingFields) > 0 {
		return errors.New(strings.Join(missingFields, " and ") + " must be set")
	}
	return nil
}
