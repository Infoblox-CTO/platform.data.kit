package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var cellContext string // --context flag for cell commands

// cellCmd is the parent command for cell management.
var cellCmd = &cobra.Command{
	Use:   "cell",
	Short: "Manage cells (infrastructure contexts)",
	Long: `Manage cells — the infrastructure contexts where packages are deployed.

A cell is a cluster-scoped Kubernetes CRD representing an isolated instance
of pipeline infrastructure. Each cell owns a namespace containing Store CRs
that resolve to physical databases, buckets, and message brokers.

Examples:
  dp cell list                          # list all cells in current cluster
  dp cell list --context k3d-dp-local   # list cells in specific cluster
  dp cell show canary                   # show cell details + stores
  dp cell stores canary                 # list stores in canary cell`,
}

// cellListCmd lists all cells in the cluster.
var cellListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cells in the cluster",
	Long: `List all Cell custom resources in the current Kubernetes cluster.

Uses kubectl to query the cluster for Cell CRDs.

Examples:
  dp cell list
  dp cell list --context k3d-dp-local
  dp cell list --context arn:aws:eks:us-east-1:...:cluster/dp-prod`,
	RunE: runCellList,
}

// cellShowCmd shows details of a specific cell.
var cellShowCmd = &cobra.Command{
	Use:   "show <cell-name>",
	Short: "Show cell details",
	Long: `Show detailed information about a specific cell, including its
namespace, labels, status, and deployed packages.

Examples:
  dp cell show canary
  dp cell show stable --context arn:aws:eks:...:dp-prod`,
	Args: cobra.ExactArgs(1),
	RunE: runCellShow,
}

// cellStoresCmd lists stores in a cell.
var cellStoresCmd = &cobra.Command{
	Use:   "stores <cell-name>",
	Short: "List stores in a cell",
	Long: `List all Store custom resources in a cell's namespace.

Stores are namespaced CRs that provide connection details for the
infrastructure in a cell. The same store name in different cells
points to different physical infrastructure.

Examples:
  dp cell stores canary
  dp cell stores stable --context arn:aws:eks:...:dp-prod`,
	Args: cobra.ExactArgs(1),
	RunE: runCellStores,
}

func init() {
	rootCmd.AddCommand(cellCmd)
	cellCmd.AddCommand(cellListCmd)
	cellCmd.AddCommand(cellShowCmd)
	cellCmd.AddCommand(cellStoresCmd)

	cellCmd.PersistentFlags().StringVar(&cellContext, "context", "", "kubectl context for multi-cluster operations")
}

func runCellList(cmd *cobra.Command, args []string) error {
	kubectlArgs := []string{"get", "cells", "-o", "json"}
	if cellContext != "" {
		kubectlArgs = append([]string{"--context", cellContext}, kubectlArgs...)
	}

	out, err := execKubectl(kubectlArgs...)
	if err != nil {
		return fmt.Errorf("failed to list cells: %w\n\nMake sure Cell CRDs are installed and you have cluster access.\nHint: dp dev up installs CRDs automatically.", err)
	}

	// Parse the JSON list.
	var list struct {
		Items []struct {
			Metadata struct {
				Name              string `json:"name"`
				CreationTimestamp string `json:"creationTimestamp"`
			} `json:"metadata"`
			Spec struct {
				Namespace string            `json:"namespace"`
				Labels    map[string]string `json:"labels"`
			} `json:"spec"`
			Status struct {
				Ready        bool  `json:"ready"`
				StoreCount   int32 `json:"storeCount"`
				PackageCount int32 `json:"packageCount"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		return fmt.Errorf("failed to parse cell list: %w", err)
	}

	if len(list.Items) == 0 {
		fmt.Println("No cells found.")
		fmt.Println("\nTo create a cell:")
		fmt.Println("  dp dev up --cell canary    # create a cell in local k3d")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tNAMESPACE\tREADY\tSTORES\tPACKAGES\tLABELS")
	for _, c := range list.Items {
		labels := formatLabels(c.Spec.Labels)
		fmt.Fprintf(w, "%s\t%s\t%v\t%d\t%d\t%s\n",
			c.Metadata.Name,
			c.Spec.Namespace,
			c.Status.Ready,
			c.Status.StoreCount,
			c.Status.PackageCount,
			labels,
		)
	}
	w.Flush()
	return nil
}

func runCellShow(cmd *cobra.Command, args []string) error {
	cellName := args[0]

	kubectlArgs := []string{"get", "cell", cellName, "-o", "json"}
	if cellContext != "" {
		kubectlArgs = append([]string{"--context", cellContext}, kubectlArgs...)
	}

	out, err := execKubectl(kubectlArgs...)
	if err != nil {
		return fmt.Errorf("cell %q not found: %w", cellName, err)
	}

	var cell struct {
		Metadata struct {
			Name              string `json:"name"`
			CreationTimestamp string `json:"creationTimestamp"`
		} `json:"metadata"`
		Spec struct {
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"spec"`
		Status struct {
			Ready        bool  `json:"ready"`
			StoreCount   int32 `json:"storeCount"`
			PackageCount int32 `json:"packageCount"`
		} `json:"status"`
	}
	if err := json.Unmarshal(out, &cell); err != nil {
		return fmt.Errorf("failed to parse cell details: %w", err)
	}

	fmt.Printf("Cell: %s\n", cell.Metadata.Name)
	fmt.Printf("  Namespace:  %s\n", cell.Spec.Namespace)
	fmt.Printf("  Ready:      %v\n", cell.Status.Ready)
	fmt.Printf("  Stores:     %d\n", cell.Status.StoreCount)
	fmt.Printf("  Packages:   %d\n", cell.Status.PackageCount)
	fmt.Printf("  Created:    %s\n", cell.Metadata.CreationTimestamp)
	if len(cell.Spec.Labels) > 0 {
		fmt.Printf("  Labels:\n")
		for k, v := range cell.Spec.Labels {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}

	// Also show stores.
	fmt.Printf("\nStores in %s:\n", cell.Spec.Namespace)
	return listStoresInNamespace(cell.Spec.Namespace)
}

func runCellStores(cmd *cobra.Command, args []string) error {
	cellName := args[0]
	ns := "dp-" + cellName
	fmt.Printf("Stores in cell %q (namespace: %s):\n\n", cellName, ns)
	return listStoresInNamespace(ns)
}

func listStoresInNamespace(ns string) error {
	kubectlArgs := []string{"get", "stores", "-n", ns, "-o", "json"}
	if cellContext != "" {
		kubectlArgs = append([]string{"--context", cellContext}, kubectlArgs...)
	}

	out, err := execKubectl(kubectlArgs...)
	if err != nil {
		return fmt.Errorf("failed to list stores in namespace %s: %w", ns, err)
	}

	var list struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Spec struct {
				Connector  string            `json:"connector"`
				Connection map[string]string `json:"connection"`
			} `json:"spec"`
			Status struct {
				Ready bool `json:"ready"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		return fmt.Errorf("failed to parse store list: %w", err)
	}

	if len(list.Items) == 0 {
		fmt.Printf("  (no stores found in %s)\n", ns)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tCONNECTOR\tREADY")
	for _, s := range list.Items {
		fmt.Fprintf(w, "  %s\t%s\t%v\n",
			s.Metadata.Name,
			s.Spec.Connector,
			s.Status.Ready,
		)
	}
	w.Flush()
	return nil
}

// execKubectl runs kubectl with the given args and returns stdout.
func execKubectl(args ...string) ([]byte, error) {
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

// formatLabels returns a compact label string.
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}
