package score

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type scoreDocument struct {
	Resources map[string]scoreResourceSpec `yaml:"resources"`
}

type scoreResourceSpec struct {
	Type string `yaml:"type"`
}

type workloadClassDocument struct {
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		ApprovedResourceTypes []string `yaml:"approvedResourceTypes"`
	} `yaml:"spec"`
}

type ValidationResult struct {
	ScorePath               string   `json:"score_path"`
	ContractPath            string   `json:"contract_path"`
	WorkloadClass           string   `json:"workload_class"`
	State                   string   `json:"state"`
	Allowed                 bool     `json:"allowed"`
	ResourceTypes           []string `json:"resource_types"`
	ApprovedResourceTypes   []string `json:"approved_resource_types"`
	UnapprovedResourceTypes []string `json:"unapproved_resource_types,omitempty"`
	Reason                  string   `json:"reason"`
}

type ValidateWorkloadOptions struct {
	ScorePath    string
	ContractPath string
}

func ValidateWorkload(opts ValidateWorkloadOptions) (*ValidationResult, error) {
	if strings.TrimSpace(opts.ScorePath) == "" {
		return nil, fmt.Errorf("score_path is required")
	}
	if strings.TrimSpace(opts.ContractPath) == "" {
		return nil, fmt.Errorf("contract_path is required")
	}

	scorePath, err := filepath.Abs(opts.ScorePath)
	if err != nil {
		return nil, fmt.Errorf("resolve score path: %w", err)
	}
	contractPath, err := filepath.Abs(opts.ContractPath)
	if err != nil {
		return nil, fmt.Errorf("resolve contract path: %w", err)
	}

	scoreData, err := os.ReadFile(scorePath)
	if err != nil {
		return nil, fmt.Errorf("read score file: %w", err)
	}
	contractData, err := os.ReadFile(contractPath)
	if err != nil {
		return nil, fmt.Errorf("read contract file: %w", err)
	}

	var scoreDoc scoreDocument
	if err := yaml.Unmarshal(scoreData, &scoreDoc); err != nil {
		return nil, fmt.Errorf("parse score file: %w", err)
	}
	var contractDoc workloadClassDocument
	if err := yaml.Unmarshal(contractData, &contractDoc); err != nil {
		return nil, fmt.Errorf("parse contract file: %w", err)
	}

	resourceTypes := collectResourceTypes(scoreDoc.Resources)
	approved := uniqueSortedStrings(contractDoc.Spec.ApprovedResourceTypes)
	if len(approved) == 0 {
		return nil, fmt.Errorf("contract %s does not declare spec.approvedResourceTypes", opts.ContractPath)
	}

	approvedSet := make(map[string]struct{}, len(approved))
	for _, item := range approved {
		approvedSet[item] = struct{}{}
	}

	unapproved := make([]string, 0)
	for _, item := range resourceTypes {
		if _, ok := approvedSet[item]; !ok {
			unapproved = append(unapproved, item)
		}
	}

	result := &ValidationResult{
		ScorePath:               scorePath,
		ContractPath:            contractPath,
		WorkloadClass:           strings.TrimSpace(contractDoc.Metadata.Name),
		ResourceTypes:           resourceTypes,
		ApprovedResourceTypes:   approved,
		UnapprovedResourceTypes: unapproved,
	}

	if len(unapproved) == 0 {
		result.State = "ALLOW"
		result.Allowed = true
		result.Reason = "All declared Score resource types are approved by the workload class."
		return result, nil
	}

	result.State = "ESCALATE"
	result.Allowed = false
	result.Reason = fmt.Sprintf(
		"Resource types %s are outside the approved workload-class allow list and require platform review.",
		strings.Join(unapproved, ", "),
	)
	return result, nil
}

func collectResourceTypes(resources map[string]scoreResourceSpec) []string {
	if len(resources) == 0 {
		return nil
	}
	types := make([]string, 0, len(resources))
	for _, resource := range resources {
		item := strings.TrimSpace(resource.Type)
		if item == "" {
			continue
		}
		types = append(types, item)
	}
	return uniqueSortedStrings(types)
}

func uniqueSortedStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
