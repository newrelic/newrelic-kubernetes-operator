package v1

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

type ChannelHeader struct {
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
	Secret    string `json:"secret,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	KeyName   string `json:"key_name,omitempty"`
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

	Payload map[string]string `json:"payload,omitempty"`

	Headers []ChannelHeader `json:"headers,omitempty"`
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

func getSecret(name types.NamespacedName, key string, k8sClient client.Client) (string, error) {
	var apiKeySecret v1.Secret

	err := k8sClient.Get(context.Background(), name, &apiKeySecret)
	if err != nil {
		return "", err
	}

	return string(apiKeySecret.Data[key]), nil
}

// APIChannel - Converts AlertsChannelSpec object to alerts.Channel
func (in AlertsChannelSpec) APIChannel(k8sClient client.Client) (alerts.Channel, error) {
	headers := in.Configuration.Headers
	in.Configuration.Headers = nil
	jsonString, _ := json.Marshal(in)

	var APIChannel alerts.Channel
	err := json.Unmarshal(jsonString, &APIChannel) // nolint
	if err != nil {
		return alerts.Channel{}, err
	}

	APIChannel.Links = alerts.ChannelLinks{}

	// Spec holds headers in a different format, need to convert to API format
	if len(headers) > 0 {
		APIChannel.Configuration.Headers = map[string]interface{}{}
		for _, header := range headers {
			if header.Value != "" {
				APIChannel.Configuration.Headers[header.Name] = header.Value
			} else {
				name := types.NamespacedName{
					Namespace: header.Namespace,
					Name:      header.Secret,
				}
				value, err := getSecret(name, header.KeyName, k8sClient)
				if err != nil {
					return alerts.Channel{}, err
				}
				APIChannel.Configuration.Headers[header.Name] = value
			}
		}
	}

	return APIChannel, nil
}
