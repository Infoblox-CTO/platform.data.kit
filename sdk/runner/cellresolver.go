// Package runner provides the cell resolver for loading Stores from k8s cells.
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// CellResolver resolves Stores from a Cell's Kubernetes namespace via kubectl.
// When a cell is specified (dk run --cell canary), Stores are fetched from the
// cell's namespace (dk-<cell>) instead of the package's local store/ directory.
type CellResolver struct {
	// CellName is the cell to resolve stores from.
	CellName string

	// KubeContext is the optional kubectl context (for multi-cluster).
	KubeContext string

	// Output is where to write status messages.
	Output io.Writer
}

// NewCellResolver creates a resolver for the given cell.
func NewCellResolver(cellName, kubeContext string, output io.Writer) *CellResolver {
	return &CellResolver{
		CellName:    cellName,
		KubeContext: kubeContext,
		Output:      output,
	}
}

// cellNamespace returns the k8s namespace for a cell.
// Convention: dk-<cellName>
func (r *CellResolver) cellNamespace() string {
	return "dk-" + r.CellName
}

// ResolveStore fetches a single Store by name from the cell's namespace.
func (r *CellResolver) ResolveStore(ctx context.Context, storeName string) (*contracts.Store, error) {
	ns := r.cellNamespace()
	args := []string{"get", "store", storeName, "-n", ns, "-o", "json"}
	if r.KubeContext != "" {
		args = append([]string{"--context", r.KubeContext}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if strings.Contains(stderr, "not found") || strings.Contains(stderr, "NotFound") {
				return nil, fmt.Errorf("store %q not found in cell %q (namespace %s)", storeName, r.CellName, ns)
			}
			return nil, fmt.Errorf("kubectl get store %s -n %s: %s", storeName, ns, stderr)
		}
		return nil, fmt.Errorf("kubectl get store %s -n %s: %w", storeName, ns, err)
	}

	return parseStoreFromJSON(out)
}

// ListStores returns all Stores in the cell's namespace.
func (r *CellResolver) ListStores(ctx context.Context) ([]*contracts.Store, error) {
	ns := r.cellNamespace()
	args := []string{"get", "stores", "-n", ns, "-o", "json"}
	if r.KubeContext != "" {
		args = append([]string{"--context", r.KubeContext}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			return nil, fmt.Errorf("kubectl get stores -n %s: %s", ns, stderr)
		}
		return nil, fmt.Errorf("kubectl get stores -n %s: %w", ns, err)
	}

	return parseStoreListFromJSON(out)
}

// CellExists checks whether the Cell CR exists in the cluster.
func (r *CellResolver) CellExists(ctx context.Context) (bool, error) {
	args := []string{"get", "cell", r.CellName, "-o", "name"}
	if r.KubeContext != "" {
		args = append([]string{"--context", r.KubeContext}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if err := cmd.Run(); err != nil {
		return false, nil
	}
	return true, nil
}

// --- Internal JSON parsing ---

// storeResource represents the k8s Store CRD JSON structure.
type storeResource struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Connector        string            `json:"connector"`
		ConnectorVersion string            `json:"connectorVersion,omitempty"`
		Connection       map[string]any    `json:"connection,omitempty"`
		Secrets          map[string]string `json:"secrets,omitempty"`
	} `json:"spec"`
}

// storeListResource represents a k8s Store list.
type storeListResource struct {
	Items []storeResource `json:"items"`
}

// parseStoreFromJSON converts a kubectl JSON output into a contracts.Store.
func parseStoreFromJSON(data []byte) (*contracts.Store, error) {
	var res storeResource
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("parsing store JSON: %w", err)
	}
	return convertStoreResource(&res), nil
}

// parseStoreListFromJSON converts a kubectl JSON list into contracts.Store slice.
func parseStoreListFromJSON(data []byte) ([]*contracts.Store, error) {
	var list storeListResource
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parsing store list JSON: %w", err)
	}
	stores := make([]*contracts.Store, len(list.Items))
	for i := range list.Items {
		stores[i] = convertStoreResource(&list.Items[i])
	}
	return stores, nil
}

// convertStoreResource converts the raw k8s JSON struct to a contracts.Store.
func convertStoreResource(res *storeResource) *contracts.Store {
	// Convert connection map[string]any to the same (contracts uses map[string]any).
	conn := make(map[string]any, len(res.Spec.Connection))
	for k, v := range res.Spec.Connection {
		conn[k] = v
	}

	return &contracts.Store{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Store",
		Metadata: contracts.StoreMetadata{
			Name:      res.Metadata.Name,
			Namespace: res.Metadata.Namespace,
		},
		Spec: contracts.StoreSpec{
			Connector:        res.Spec.Connector,
			ConnectorVersion: res.Spec.ConnectorVersion,
			Connection:       conn,
			Secrets:          res.Spec.Secrets,
		},
	}
}
