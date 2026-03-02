package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CellSpec defines the desired state of a Cell.
type CellSpec struct {
	// Namespace is the Kubernetes namespace where this cell's workloads
	// and Stores are deployed (e.g., "dk-canary").
	Namespace string `json:"namespace"`

	// Labels are key-value metadata for cell selection and filtering.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// CellStatus defines the observed state of a Cell.
type CellStatus struct {
	// Ready indicates whether the cell's namespace and stores are initialised.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// StoreCount is the number of Store CRs in the cell's namespace.
	// +optional
	StoreCount int32 `json:"storeCount,omitempty"`

	// PackageCount is the number of PackageDeployments targeting this cell.
	// +optional
	PackageCount int32 `json:"packageCount,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Stores",type=integer,JSONPath=`.status.storeCount`
// +kubebuilder:printcolumn:name="Packages",type=integer,JSONPath=`.status.packageCount`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Cell represents an isolated infrastructure context for deploying data packages.
// Cells are cluster-scoped — they answer the question "where does a Store name
// resolve to?" by owning a namespace that contains Store CRs.
type Cell struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CellSpec   `json:"spec,omitempty"`
	Status CellStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CellList contains a list of Cell.
type CellList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cell `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cell{}, &CellList{})
}
