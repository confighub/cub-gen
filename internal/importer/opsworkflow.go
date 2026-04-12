package importer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

type opsWorkflowHints struct {
	BaseSpecPath    string
	OverlaySpecPath string
}

func opsWorkflowPathHintsFromInputs(inputs []string) opsWorkflowHints {
	h := opsWorkflowHints{
		BaseSpecPath: registry.HintDefault(model.GeneratorOpsFlow, "base_spec_path", "operations.yaml"),
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch {
		case base == "operations.yaml" || base == "operations.yml" || base == "workflow.yaml" || base == "workflow.yml":
			h.BaseSpecPath = p
		case strings.HasPrefix(base, "operations-") || strings.HasPrefix(base, "workflow-"):
			if h.OverlaySpecPath == "" || p < h.OverlaySpecPath {
				h.OverlaySpecPath = p
			}
		}
	}
	return h
}

type opsWorkflowDoc struct {
	Path         string
	WorkflowName string
	Schedule     string
	ActionNames  []string
}

type opsExecutionPolicy struct {
	Path           string
	AllowedActions []string
	BlockedActions []string
	ApprovalGates  []string
}

func opsWorkflowAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.OpsWorkflowAnalysis {
	if g.Kind != model.GeneratorOpsFlow {
		return nil
	}

	workflowPaths := opsWorkflowPathsFromInputs(g.Inputs)
	if len(workflowPaths) == 0 {
		return nil
	}

	docs := make([]opsWorkflowDoc, 0, len(workflowPaths))
	for _, path := range workflowPaths {
		doc, err := parseOpsWorkflowFile(detection.Repo, path)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil
	}

	baseDoc := opsBaseWorkflowDoc(docs)
	if baseDoc.Path == "" {
		baseDoc = docs[0]
	}

	policyPath := opsExecutionPolicyPathFromRepo(detection.Repo, g.Inputs)
	policy := opsExecutionPolicy{}
	if policyPath != "" {
		if parsed, err := parseOpsExecutionPolicyFile(detection.Repo, policyPath); err == nil {
			policy = parsed
		}
	}

	workflowPathValues := make([]string, 0, len(docs))
	overlayPaths := make([]string, 0, len(docs))
	workflowNames := make([]string, 0, len(docs))
	schedules := make([]string, 0, len(docs))
	scheduleOverrides := make([]string, 0)
	actionSet := map[string]struct{}{}
	addedActions := make([]string, 0)
	removedActions := make([]string, 0)

	for _, doc := range docs {
		workflowPathValues = append(workflowPathValues, doc.Path)
		if doc.Path != baseDoc.Path {
			overlayPaths = append(overlayPaths, doc.Path)
		}
		if doc.WorkflowName != "" {
			workflowNames = append(workflowNames, doc.WorkflowName)
		}
		if doc.Schedule != "" {
			schedules = append(schedules, doc.Schedule)
		}
		if doc.Path != baseDoc.Path && doc.Schedule != "" && doc.Schedule != baseDoc.Schedule {
			scheduleOverrides = append(scheduleOverrides, doc.Path+":"+doc.Schedule)
		}
		for _, action := range doc.ActionNames {
			actionSet[action] = struct{}{}
		}
		if doc.Path != baseDoc.Path {
			addedActions = append(addedActions, differenceStrings(doc.ActionNames, baseDoc.ActionNames)...)
			removedActions = append(removedActions, differenceStrings(baseDoc.ActionNames, doc.ActionNames)...)
		}
	}

	actionNames := sortedStringSet(actionSet)
	allowedActions := uniqueSortedStrings(policy.AllowedActions)
	blockedActions := uniqueSortedStrings(policy.BlockedActions)
	unapprovedActions := differenceAgainstAllowList(actionNames, allowedActions)
	blockedActionsUsed := intersectionStrings(actionNames, blockedActions)

	return &model.OpsWorkflowAnalysis{
		WorkflowPaths:        uniqueSortedStrings(workflowPathValues),
		BaseWorkflowPath:     baseDoc.Path,
		OverlayWorkflowPaths: uniqueSortedStrings(overlayPaths),
		PolicyPath:           policy.Path,
		WorkflowNames:        uniqueSortedStrings(workflowNames),
		Schedules:            uniqueSortedStrings(schedules),
		ScheduleOverrides:    uniqueSortedStrings(scheduleOverrides),
		ActionNames:          actionNames,
		AllowedActions:       allowedActions,
		BlockedActions:       blockedActions,
		ApprovalGates:        uniqueSortedStrings(policy.ApprovalGates),
		UnapprovedActions:    unapprovedActions,
		BlockedActionsUsed:   blockedActionsUsed,
		AddedActions:         uniqueSortedStrings(addedActions),
		RemovedActions:       uniqueSortedStrings(removedActions),
	}
}

func opsWorkflowPathsFromInputs(inputs []string) []string {
	paths := make([]string, 0, len(inputs))
	for _, in := range inputs {
		base := strings.ToLower(filepath.Base(in))
		ext := strings.ToLower(filepath.Ext(base))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if !strings.HasPrefix(base, "operations") && !strings.HasPrefix(base, "workflow") {
			continue
		}
		paths = append(paths, filepath.ToSlash(in))
	}
	return uniqueSortedStrings(paths)
}

func opsBaseWorkflowDoc(docs []opsWorkflowDoc) opsWorkflowDoc {
	for _, doc := range docs {
		base := strings.ToLower(filepath.Base(doc.Path))
		if base == "operations.yaml" || base == "operations.yml" || base == "workflow.yaml" || base == "workflow.yml" {
			return doc
		}
	}
	return opsWorkflowDoc{}
}

func opsExecutionPolicyPathFromRepo(repo string, inputs []string) string {
	knownBasenames := map[string]struct{}{
		"execution-policy.yaml": {},
		"execution-policy.yml":  {},
		"workflow-policy.yaml":  {},
		"workflow-policy.yml":   {},
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		if _, ok := knownBasenames[base]; !ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(p))); err == nil {
			return p
		}
	}

	candidates := []string{
		"platform/execution-policy.yaml",
		"platform/execution-policy.yml",
		"execution-policy.yaml",
		"execution-policy.yml",
		"platform/workflow-policy.yaml",
		"platform/workflow-policy.yml",
		"workflow-policy.yaml",
		"workflow-policy.yml",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(candidate))); err == nil {
			return candidate
		}
	}
	return ""
}

func parseOpsWorkflowFile(repo, path string) (opsWorkflowDoc, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return opsWorkflowDoc{}, err
	}

	doc := opsWorkflowDoc{Path: filepath.ToSlash(path)}
	actionSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inWorkflow := false
	workflowIndent := 0
	inTriggers := false
	triggersIndent := 0
	inActions := false
	actionsIndent := 0

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if inWorkflow && indent <= workflowIndent && !strings.HasPrefix(trimmed, "workflow:") {
			inWorkflow = false
		}
		if inTriggers && indent <= triggersIndent && !strings.HasPrefix(trimmed, "triggers:") {
			inTriggers = false
		}
		if inActions && indent <= actionsIndent && !strings.HasPrefix(trimmed, "actions:") {
			inActions = false
		}

		if strings.HasPrefix(trimmed, "workflow:") {
			inWorkflow = true
			workflowIndent = indent
			continue
		}
		if strings.HasPrefix(trimmed, "triggers:") {
			inTriggers = true
			triggersIndent = indent
			continue
		}
		if strings.HasPrefix(trimmed, "actions:") {
			inActions = true
			actionsIndent = indent
			continue
		}

		if inWorkflow && strings.HasPrefix(trimmed, "name:") && indent >= workflowIndent+2 {
			doc.WorkflowName = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")))
			continue
		}
		if inTriggers && strings.HasPrefix(trimmed, "schedule:") && indent >= triggersIndent+2 {
			doc.Schedule = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "schedule:")))
			continue
		}
		if inActions && indent == actionsIndent+2 && strings.HasSuffix(trimmed, ":") {
			action := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			if action != "" {
				actionSet[action] = struct{}{}
			}
			continue
		}
	}

	doc.ActionNames = sortedStringSet(actionSet)
	return doc, nil
}

func parseOpsExecutionPolicyFile(repo, path string) (opsExecutionPolicy, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return opsExecutionPolicy{}, err
	}

	policy := opsExecutionPolicy{Path: filepath.ToSlash(path)}
	allowedSet := map[string]struct{}{}
	blockedSet := map[string]struct{}{}
	approvalSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inSpec := false
	specIndent := 0
	mode := ""
	currentApprovalEnv := ""
	currentApprovalCount := ""

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if strings.HasPrefix(trimmed, "spec:") {
			inSpec = true
			specIndent = indent
			mode = ""
			currentApprovalEnv = ""
			currentApprovalCount = ""
			continue
		}
		if !inSpec {
			continue
		}
		if indent <= specIndent {
			inSpec = false
			mode = ""
			currentApprovalEnv = ""
			currentApprovalCount = ""
			continue
		}

		if indent == specIndent+2 && strings.Contains(trimmed, ":") {
			if mode == "approval_gates" && currentApprovalEnv != "" && currentApprovalCount != "" {
				approvalSet[currentApprovalEnv+":"+currentApprovalCount] = struct{}{}
			}
			parts := strings.SplitN(trimmed, ":", 2)
			mode = strings.TrimSpace(parts[0])
			value := ""
			if len(parts) == 2 {
				value = strings.TrimSpace(parts[1])
			}
			currentApprovalEnv = ""
			currentApprovalCount = ""

			switch mode {
			case "allowed_actions":
				for _, item := range parseYAMLInlineList(value) {
					allowedSet[item] = struct{}{}
				}
			case "blocked_actions":
				for _, item := range parseYAMLInlineList(value) {
					blockedSet[item] = struct{}{}
				}
			}
			continue
		}

		switch mode {
		case "allowed_actions":
			if strings.HasPrefix(trimmed, "- ") {
				value := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if value != "" {
					allowedSet[value] = struct{}{}
				}
			}
		case "blocked_actions":
			if strings.HasPrefix(trimmed, "- ") {
				value := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if value != "" {
					blockedSet[value] = struct{}{}
				}
			}
		case "approval_gates":
			if indent == specIndent+4 && strings.HasSuffix(trimmed, ":") {
				if currentApprovalEnv != "" && currentApprovalCount != "" {
					approvalSet[currentApprovalEnv+":"+currentApprovalCount] = struct{}{}
				}
				currentApprovalEnv = parseYAMLScalar(strings.TrimSuffix(trimmed, ":"))
				currentApprovalCount = ""
				continue
			}
			if currentApprovalEnv != "" && strings.HasPrefix(trimmed, "required_approvals:") {
				currentApprovalCount = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "required_approvals:")))
			}
		}
	}
	if currentApprovalEnv != "" && currentApprovalCount != "" {
		approvalSet[currentApprovalEnv+":"+currentApprovalCount] = struct{}{}
	}

	policy.AllowedActions = sortedStringSet(allowedSet)
	policy.BlockedActions = sortedStringSet(blockedSet)
	policy.ApprovalGates = sortedStringSet(approvalSet)
	return policy, nil
}
