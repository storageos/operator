package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OperatorConfig is the Schema for the operatorconfigs API
type OperatorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	// WebhookCertRefreshInterval is the interval at which the webhook cert is
	// refreshed.
	WebhookCertRefreshInterval *metav1.Duration `json:"webhookCertRefreshInterval,omitempty"`

	// WebhookServiceName is the service name of the webhook server.
	WebhookServiceName string `json:"webhookServiceName,omitempty"`

	// WebhookSecretRef is the secret reference that stores webhook secrets.
	WebhookSecretRef string `json:"webhookSecretRef,omitempty"`

	// ValidatingWebhookConfigRef is the reference of the validating webhook
	// configuration.
	ValidatingWebhookConfigRef string `json:"validatingWebhookConfigRef,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OperatorConfig{})
}
