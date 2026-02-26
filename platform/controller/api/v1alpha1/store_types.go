package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StoreSpec defines the desired state of a Store.
type StoreSpec struct {
	// Connector references the provider identity of the Connector this
	// Store uses (e.g., "postgres", "s3"). Maps to spec.provider on the
	// Connector CR.
	Connector string `json:"connector"`

	// ConnectorVersion is an optional semver range constraint that pins
	// this Store to a compatible Connector version (e.g., "^1.0.0").
	// +optional
	ConnectorVersion string `json:"connectorVersion,omitempty"`

	// Connection contains non-secret connection parameters
	// (e.g., connection_string, bucket, endpoint, region).
	// +optional
	Connection map[string]string `json:"connection,omitempty"`

	// Secrets contains secret-bearing connection parameters.
	// Values may use ${VAR} interpolation resolved from Kubernetes Secrets
	// in the same namespace.
	// +optional
	Secrets map[string]string `json:"secrets,omitempty"`
}

// StoreStatus defines the observed state of a Store.
type StoreStatus struct {
	// Ready indicates whether the store's connection has been verified.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Connector",type=string,JSONPath=`.spec.connector`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Store represents a named infrastructure endpoint within a Cell's namespace.
// The same Store name in different Cell namespaces resolves to different
// physical infrastructure (e.g., different databases, buckets, or topic prefixes).
type Store struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StoreSpec   `json:"spec,omitempty"`
	Status StoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StoreList contains a list of Store.
type StoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Store `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Store{}, &StoreList{})
}
