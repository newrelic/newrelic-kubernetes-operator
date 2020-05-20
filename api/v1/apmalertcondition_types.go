package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApmAlertConditionSpec defines the desired state of NrqlAlertCondition
type ApmAlertConditionSpec struct {
	Terms               []AlertConditionTerm        `json:"terms,omitempty"`
	Type                string                      `json:"type,omitempty"` //TODO: add conditionType or pull from alerts package or make string
	Name                string                      `json:"name,omitempty"`
	RunbookURL          string                      `json:"runbook_url,omitempty"`
	Metric              string                      `json:"metric,omitempty"` //TODO: check type
	UserDefined         alerts.ConditionUserDefined `json:"user_defined,omitempty"`
	Scope               string                      `json:"condition_scope,omitempty"`
	Entities            []string                    `json:"entities,omitempty"`
	GCMetric            string                      `json:"gc_metric,omitempty"`
	PolicyID            int                         `json:"-"`
	ID                  int                         `json:"id,omitempty"`
	ViolationCloseTimer int                         `json:"violation_close_timer,omitempty"`
	Enabled             bool                        `json:"enabled"`
	ExistingPolicyID    int                         `json:"existing_policy_id,omitempty"`
	APIKey              string                      `json:"api_key,omitempty"`
	APIKeySecret        NewRelicAPIKeySecret        `json:"api_key_secret,omitempty"`
	Region              string                      `json:"region,omitempty"`
}

/* going to need to address these metrictypes with conditions
AjaxResponseTime:       "ajax_response_time",
AjaxThroughput:         "ajax_throughput",
Apdex:                  "apdex",
CPUPercentage:          "cpu_percentage",
Database:               "database",
DiskIOPercentage:       "disk_io_percentage",
DomProcessing:          "dom_processing",
EndUserApdex:           "end_user_apdex",
ErrorCount:             "error_count",
ErrorPercentage:        "error_percentage",
FullestDiskPercentage:  "fullest_disk_percentage",
Images:                 "images",
JSON:                   "json",
LoadAverageOneMinute:   "load_average_one_minute",
MemoryPercentage:       "memory_percentage",
MobileCrashRate:        "mobile_crash_rate",
Network:                "network",
NetworkErrorPercentage: "network_error_percentage",
PageRendering:          "page_rendering",
PageViewThroughput:     "page_view_throughput",
PageViewsWithJsErrors:  "page_views_with_js_errors",
RequestQueuing:         "request_queuing",
ResponseTime:           "response_time",
ResponseTimeBackground: "response_time_background",
ResponseTimeWeb:        "response_time_web",
StatusErrorPercentage:  "status_error_percentage",
Throughput:             "throughput",
ThroughputBackground:   "throughput_background",
ThroughputWeb:          "throughput_web",
TotalPageLoad:          "total_page_load",
UserDefined:            "user_defined",
ViewLoading:            "view_loading",
WebApplication:         "web_application",
*/

// ApmAlertConditionStatus defines the observed state of ApmAlertCondition
type ApmAlertConditionStatus struct {
	AppliedSpec *ApmAlertConditionSpec `json:"applied_spec"`
	ConditionID int                    `json:"condition_id"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// ApmAlertCondition is the Schema for the apmalertconditions API
type ApmAlertCondition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApmAlertConditionSpec   `json:"spec,omitempty"`
	Status ApmAlertConditionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NrqlAlertConditionList contains a list of NrqlAlertCondition
type ApmAlertConditionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApmAlertCondition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApmAlertCondition{}, &ApmAlertConditionList{})
}

func (in ApmAlertConditionSpec) APICondition() alerts.Condition {
	jsonString, _ := json.Marshal(in)
	var APICondition alerts.Condition
	json.Unmarshal(jsonString, &APICondition) //nolint
	//APICondition.PolicyID = spec.ExistingPolicyId

	return APICondition
}
