package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	componentbaseversion "k8s.io/component-base/version"

	"github.com/openshift/origin/test/extended/apiserver/inventory"
)

func main() {
	cmd := &cobra.Command{
		Use:   "write-kube-api-inventory",
		Short: "Generate Kubernetes API inventory for the current vendored Kubernetes version",
		RunE:  run,
	}

	cmd.Flags().Bool("verify", false, "Verify that the generated file matches what's on disk (exit 1 if different)")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	verify, _ := cmd.Flags().GetBool("verify")

	// Get the current Kubernetes version from component-base
	// DefaultKubeBinaryVersion is like "1.36"
	kubeVersion, err := utilversion.ParseGeneric(componentbaseversion.DefaultKubeBinaryVersion)
	if err != nil {
		return fmt.Errorf("failed to parse Kubernetes version %q: %w", componentbaseversion.DefaultKubeBinaryVersion, err)
	}

	minor := int(kubeVersion.Minor())
	fmt.Printf("Generating Kubernetes API inventory for version 1.%d\n", minor)

	// Generate the inventory
	entries, err := inventory.GenerateKubernetesInventory()
	if err != nil {
		return fmt.Errorf("failed to generate inventory: %w", err)
	}

	fmt.Printf("Generated %d Kubernetes API entries\n", len(entries))

	// Format as Go code
	goCode, err := inventory.FormatGoCode(entries, minor)
	if err != nil {
		return fmt.Errorf("failed to format Go code: %w", err)
	}

	// Determine output file path
	// This command is at test/extended/apiserver/inventory/write-kube-api-inventory/
	// Output should be at test/extended/apiserver/inventory/zz_generated_kubernetes.go
	outputPath := filepath.Join("test", "extended", "apiserver", "inventory", "zz_generated_kubernetes.go")

	if verify {
		// Verify mode: check if the file matches
		existing, err := os.ReadFile(outputPath)
		if err != nil {
			return fmt.Errorf("failed to read existing file for verification: %w", err)
		}

		if !bytes.Equal(existing, []byte(goCode)) {
			fmt.Fprintf(os.Stderr, "ERROR: Generated file does not match %s\n", outputPath)
			fmt.Fprintf(os.Stderr, "Run: make update-kube-api-inventory\n")
			return fmt.Errorf("verification failed")
		}

		fmt.Printf("Verification passed: %s is up to date\n", outputPath)
		return nil
	}

	// Write mode: write the generated file
	if err := os.WriteFile(outputPath, []byte(goCode), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Successfully wrote %s\n", outputPath)
	return nil
}
