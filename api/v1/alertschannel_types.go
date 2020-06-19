package v1

import (
	"encoding/json"

	"github.com/newrelic/newrelic-client-go/pkg/alerts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AlertsChannelSpec defines the desired state of AlertsChannel
type AlertsChannelSpec struct {
	ID            int                        `json:"id,omitempty"`
	Name          string                     `json:"name"`
	APIKey        string                     `json:"api_key,omitempty"`
	APIKeySecret  NewRelicAPIKeySecret       `json:"api_key_secret,omitempty"`
	Region        string                     `json:"region,omitempty"`
	Type          string                     `json:"type,omitempty"`
	Links         ChannelLinks               `json:"links,omitempty"`
	Configuration AlertsChannelConfiguration `json:"configuration,omitempty"`
}

// ChannelLinks - copy of alerts.ChannelLinks
type ChannelLinks struct {
	PolicyIDs               []int               `json:"policy_ids,omitempty"`
	PolicyNames             []string            `json:"policy_names,omitempty"`
	PolicyKubernetesObjects []metav1.ObjectMeta `json:"policy_kubernetes_objects,omitempty"`
}

// AlertsChannelStatus defines the observed state of AlertsChannel
type AlertsChannelStatus struct {
	AppliedSpec      *AlertsChannelSpec `json:"applied_spec"`
	ChannelID        int                `json:"channel_id"`
	AppliedPolicyIDs []int              `json:"appliedPolicyIDs"`
}

// AlertsChannelConfiguration - copy of alerts.ChannelConfiguration
type AlertsChannelConfiguration struct {
	Recipients            string `json:"recipients,omitempty"`
	IncludeJSONAttachment string `json:"include_json_attachment,omitempty"`
	AuthToken             string `json:"auth_token,omitempty"`
	APIKey                string `json:"api_key,omitempty"`
	Teams                 string `json:"teams,omitempty"`
	Tags                  string `json:"tags,omitempty"`
	URL                   string `json:"url,omitempty"`
	Channel               string `json:"channel,omitempty"`
	Key                   string `json:"key,omitempty"`
	RouteKey              string `json:"route_key,omitempty"`
	ServiceKey            string `json:"service_key,omitempty"`
	BaseURL               string `json:"base_url,omitempty"`
	AuthUsername          string `json:"auth_username,omitempty"`
	AuthPassword          string `json:"auth_password,omitempty"`
	PayloadType           string `json:"payload_type,omitempty"`
	Region                string `json:"region,omitempty"`
	UserID                string `json:"user_id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Created",type="boolean",JSONPath=".status.created"

// AlertsChannel is the Schema for the AlertsChannel API
type AlertsChannel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertsChannelSpec   `json:"spec,omitempty"`
	Status AlertsChannelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AlertsChannelList contains a list of AlertsChannel
type AlertsChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertsChannel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertsChannel{}, &AlertsChannelList{})
}

// APIChannel - Converts AlertsChannelSpec object to alerts.Channel
func (in AlertsChannelSpec) APIChannel() alerts.Channel {
	jsonString, _ := json.Marshal(in)
	var APIChannel alerts.Channel
	json.Unmarshal(jsonString, &APIChannel) //nolint
	APIChannel.Links = alerts.ChannelLinks{}
	return APIChannel
}
