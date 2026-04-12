package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/confighub/cub-gen/internal/registry"
)

type swampHints struct {
	BaseConfigPath string
	WorkflowPath   string
}

func swampPathHintsFromInputs(inputs []string) swampHints {
	h := swampHints{
		BaseConfigPath: registry.HintDefault(model.GeneratorSwamp, "base_config_path", ".swamp.yaml"),
	}
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		switch {
		case base == ".swamp.yaml" || base == ".swamp.yml" || base == ".swamp.json":
			h.BaseConfigPath = p
		case strings.HasPrefix(base, "workflow-"):
			if h.WorkflowPath == "" || p < h.WorkflowPath {
				h.WorkflowPath = p
			}
		}
	}
	return h
}

type swampWorkflowDoc struct {
	Path            string
	StepNames       []string
	ModelRefs       []string
	MethodRefs      []string
	ModelMethodRefs []string
	JobStepCounts   map[string]int
}

type swampPolicy struct {
	Path                 string
	ApprovedModels       []string
	ApprovedModelMethods []string
	RequiredSteps        []string
	ForbiddenStepNames   []string
	MaxStepsPerJob       int
	MaxParallelJobs      int
}

func swampWorkflowAnalysisForGenerator(detection model.DetectionResult, g model.GeneratorDetection) *model.SwampWorkflowAnalysis {
	if g.Kind != model.GeneratorSwamp {
		return nil
	}

	workflowPaths := swampWorkflowPathsFromInputs(g.Inputs)
	if len(workflowPaths) == 0 {
		return nil
	}

	docs := make([]swampWorkflowDoc, 0, len(workflowPaths))
	for _, path := range workflowPaths {
		doc, err := parseSwampWorkflowFile(detection.Repo, path)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	if len(docs) == 0 {
		return nil
	}

	policyPath := swampPolicyPathFromRepo(detection.Repo, g.Inputs)
	policy := swampPolicy{}
	if policyPath != "" {
		if parsed, err := parseSwampPolicyFile(detection.Repo, policyPath); err == nil {
			policy = parsed
		}
	}

	stepSet := map[string]struct{}{}
	modelSet := map[string]struct{}{}
	methodSet := map[string]struct{}{}
	modelMethodSet := map[string]struct{}{}
	jobsExceedingMax := make([]string, 0)

	totalJobs := 0
	totalSteps := 0
	maxJobsInWorkflow := 0

	for _, doc := range docs {
		totalJobs += len(doc.JobStepCounts)
		totalSteps += len(doc.StepNames)
		if len(doc.JobStepCounts) > maxJobsInWorkflow {
			maxJobsInWorkflow = len(doc.JobStepCounts)
		}

		for _, step := range doc.StepNames {
			stepSet[step] = struct{}{}
		}
		for _, modelRef := range doc.ModelRefs {
			modelSet[modelRef] = struct{}{}
		}
		for _, methodRef := range doc.MethodRefs {
			methodSet[methodRef] = struct{}{}
		}
		for _, modelMethodRef := range doc.ModelMethodRefs {
			modelMethodSet[modelMethodRef] = struct{}{}
		}

		if policy.MaxStepsPerJob > 0 {
			jobNames := make([]string, 0, len(doc.JobStepCounts))
			for jobName := range doc.JobStepCounts {
				jobNames = append(jobNames, jobName)
			}
			sort.Strings(jobNames)
			for _, jobName := range jobNames {
				stepCount := doc.JobStepCounts[jobName]
				if stepCount <= policy.MaxStepsPerJob {
					continue
				}
				jobsExceedingMax = append(jobsExceedingMax, fmt.Sprintf("%s:%s(%d)", doc.Path, jobName, stepCount))
			}
		}
	}

	stepNames := sortedStringSet(stepSet)
	modelRefs := sortedStringSet(modelSet)
	methodRefs := sortedStringSet(methodSet)
	modelMethodRefs := sortedStringSet(modelMethodSet)

	requiredSteps := uniqueSortedStrings(policy.RequiredSteps)
	approvedModels := uniqueSortedStrings(policy.ApprovedModels)
	approvedModelMethods := uniqueSortedStrings(policy.ApprovedModelMethods)
	forbiddenStepNames := uniqueSortedStrings(policy.ForbiddenStepNames)
	jobsExceedingMax = uniqueSortedStrings(jobsExceedingMax)

	missingRequiredSteps := differenceStrings(requiredSteps, stepNames)
	forbiddenStepsPresent := intersectionStrings(stepNames, forbiddenStepNames)
	unapprovedModels := differenceAgainstAllowList(modelRefs, approvedModels)
	unapprovedModelMethods := differenceAgainstAllowList(modelMethodRefs, approvedModelMethods)

	baseDoc := docs[0]
	addedSteps := make([]string, 0)
	removedSteps := make([]string, 0)
	addedModelMethods := make([]string, 0)
	removedModelMethods := make([]string, 0)
	for _, doc := range docs[1:] {
		addedSteps = append(addedSteps, differenceStrings(doc.StepNames, baseDoc.StepNames)...)
		removedSteps = append(removedSteps, differenceStrings(baseDoc.StepNames, doc.StepNames)...)
		addedModelMethods = append(addedModelMethods, differenceStrings(doc.ModelMethodRefs, baseDoc.ModelMethodRefs)...)
		removedModelMethods = append(removedModelMethods, differenceStrings(baseDoc.ModelMethodRefs, doc.ModelMethodRefs)...)
	}

	workflowPathValues := make([]string, 0, len(docs))
	for _, doc := range docs {
		workflowPathValues = append(workflowPathValues, doc.Path)
	}

	return &model.SwampWorkflowAnalysis{
		WorkflowPaths:          uniqueSortedStrings(workflowPathValues),
		BaseWorkflowPath:       baseDoc.Path,
		PolicyPath:             policy.Path,
		StepNames:              stepNames,
		ModelRefs:              modelRefs,
		MethodRefs:             methodRefs,
		ModelMethodRefs:        modelMethodRefs,
		ApprovedModels:         approvedModels,
		ApprovedModelMethods:   approvedModelMethods,
		RequiredSteps:          requiredSteps,
		MissingRequiredSteps:   missingRequiredSteps,
		UnapprovedModels:       unapprovedModels,
		UnapprovedModelMethods: unapprovedModelMethods,
		ForbiddenStepNames:     forbiddenStepNames,
		ForbiddenStepsPresent:  forbiddenStepsPresent,
		MaxStepsPerJob:         policy.MaxStepsPerJob,
		MaxParallelJobs:        policy.MaxParallelJobs,
		TotalJobs:              totalJobs,
		TotalSteps:             totalSteps,
		JobsExceedingMaxSteps:  jobsExceedingMax,
		ExceedsMaxParallelJobs: policy.MaxParallelJobs > 0 && maxJobsInWorkflow > policy.MaxParallelJobs,
		AddedSteps:             uniqueSortedStrings(addedSteps),
		RemovedSteps:           uniqueSortedStrings(removedSteps),
		AddedModelMethodRefs:   uniqueSortedStrings(addedModelMethods),
		RemovedModelMethodRefs: uniqueSortedStrings(removedModelMethods),
	}
}

func swampWorkflowPathsFromInputs(inputs []string) []string {
	paths := make([]string, 0, len(inputs))
	for _, in := range inputs {
		p := filepath.ToSlash(in)
		base := strings.ToLower(filepath.Base(in))
		ext := strings.ToLower(filepath.Ext(base))
		if !strings.HasPrefix(base, "workflow-") {
			continue
		}
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		paths = append(paths, p)
	}
	return uniqueSortedStrings(paths)
}

func swampPolicyPathFromRepo(repo string, inputs []string) string {
	knownBasenames := map[string]struct{}{
		"swamp-constraints.yaml": {},
		"swamp-constraints.yml":  {},
		"workflow-policy.yaml":   {},
		"workflow-policy.yml":    {},
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
		"platform/swamp-constraints.yaml",
		"platform/swamp-constraints.yml",
		"swamp-constraints.yaml",
		"swamp-constraints.yml",
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

func parseSwampWorkflowFile(repo, path string) (swampWorkflowDoc, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return swampWorkflowDoc{}, err
	}

	doc := swampWorkflowDoc{
		Path:          filepath.ToSlash(path),
		JobStepCounts: map[string]int{},
	}
	stepSet := map[string]struct{}{}
	modelSet := map[string]struct{}{}
	methodSet := map[string]struct{}{}
	modelMethodSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inJobs := false
	jobsIndent := 0
	inSteps := false
	stepsIndent := 0
	currentJob := ""
	inTask := false
	taskIndent := 0
	taskModel := ""
	taskMethod := ""

	finalizeTask := func() {
		if taskModel != "" && taskMethod != "" {
			modelMethodSet[taskModel+"."+taskMethod] = struct{}{}
		}
		taskModel = ""
		taskMethod = ""
	}

	for _, line := range lines {
		raw := strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if inTask && indent <= taskIndent && !strings.HasPrefix(trimmed, "task:") {
			finalizeTask()
			inTask = false
		}
		if inSteps && indent <= stepsIndent && !strings.HasPrefix(trimmed, "steps:") {
			inSteps = false
		}
		if inJobs && indent <= jobsIndent && !strings.HasPrefix(trimmed, "jobs:") {
			inJobs = false
			currentJob = ""
		}

		if strings.HasPrefix(trimmed, "jobs:") {
			inJobs = true
			jobsIndent = indent
			continue
		}

		if inJobs && indent == jobsIndent+2 && strings.HasPrefix(trimmed, "- name:") {
			currentJob = parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))
			if currentJob == "" {
				currentJob = fmt.Sprintf("job-%d", len(doc.JobStepCounts)+1)
			}
			if _, ok := doc.JobStepCounts[currentJob]; !ok {
				doc.JobStepCounts[currentJob] = 0
			}
			continue
		}

		if inJobs && strings.HasPrefix(trimmed, "steps:") {
			inSteps = true
			stepsIndent = indent
			continue
		}

		if inSteps {
			if strings.HasPrefix(trimmed, "- name:") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))
				if step != "" {
					stepSet[step] = struct{}{}
					if currentJob != "" {
						doc.JobStepCounts[currentJob]++
					}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "name:") && indent >= stepsIndent+2 {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")))
				if step != "" {
					stepSet[step] = struct{}{}
					if currentJob != "" {
						doc.JobStepCounts[currentJob]++
					}
				}
				continue
			}
		}

		if strings.HasPrefix(trimmed, "task:") {
			inTask = true
			taskIndent = indent
			taskModel = ""
			taskMethod = ""
			continue
		}

		if inTask && strings.HasPrefix(trimmed, "modelIdOrName:") {
			modelRef := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "modelIdOrName:")))
			if modelRef != "" {
				modelSet[modelRef] = struct{}{}
				taskModel = modelRef
			}
			continue
		}
		if inTask && strings.HasPrefix(trimmed, "methodName:") {
			methodRef := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "methodName:")))
			if methodRef != "" {
				methodSet[methodRef] = struct{}{}
				taskMethod = methodRef
			}
			continue
		}
	}

	if inTask {
		finalizeTask()
	}

	doc.StepNames = sortedStringSet(stepSet)
	doc.ModelRefs = sortedStringSet(modelSet)
	doc.MethodRefs = sortedStringSet(methodSet)
	doc.ModelMethodRefs = sortedStringSet(modelMethodSet)
	return doc, nil
}

func parseSwampPolicyFile(repo, path string) (swampPolicy, error) {
	content, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(path)))
	if err != nil {
		return swampPolicy{}, err
	}

	policy := swampPolicy{
		Path: filepath.ToSlash(path),
	}
	approvedModelSet := map[string]struct{}{}
	approvedModelMethodSet := map[string]struct{}{}
	requiredStepSet := map[string]struct{}{}
	forbiddenStepSet := map[string]struct{}{}

	lines := strings.Split(string(content), "\n")
	inSpec := false
	specIndent := 0
	mode := ""
	currentModelMethodsKey := ""

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
			currentModelMethodsKey = ""
			continue
		}
		if !inSpec {
			continue
		}
		if indent <= specIndent {
			inSpec = false
			mode = ""
			currentModelMethodsKey = ""
			continue
		}

		if indent == specIndent+2 && strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			mode = strings.TrimSpace(parts[0])
			value := ""
			if len(parts) == 2 {
				value = strings.TrimSpace(parts[1])
			}
			currentModelMethodsKey = ""

			switch mode {
			case "max_steps_per_job":
				if parsed, parseErr := strconv.Atoi(parseYAMLScalar(value)); parseErr == nil {
					policy.MaxStepsPerJob = parsed
				}
			case "max_parallel_jobs":
				if parsed, parseErr := strconv.Atoi(parseYAMLScalar(value)); parseErr == nil {
					policy.MaxParallelJobs = parsed
				}
			case "approved_models":
				for _, item := range parseYAMLInlineList(value) {
					if item == "" {
						continue
					}
					approvedModelSet[item] = struct{}{}
				}
			case "forbidden_step_names":
				for _, item := range parseYAMLInlineList(value) {
					if item == "" {
						continue
					}
					forbiddenStepSet[item] = struct{}{}
				}
			case "required_steps":
				for _, item := range parseYAMLInlineList(value) {
					if item == "" {
						continue
					}
					requiredStepSet[item] = struct{}{}
				}
			}
			continue
		}

		switch mode {
		case "approved_models":
			if strings.HasPrefix(trimmed, "- ") {
				modelRef := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if modelRef != "" {
					approvedModelSet[modelRef] = struct{}{}
				}
			}
		case "approved_model_methods":
			if indent == specIndent+4 && strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				currentModelMethodsKey = parseYAMLScalar(parts[0])
				inline := ""
				if len(parts) == 2 {
					inline = strings.TrimSpace(parts[1])
				}
				for _, method := range parseYAMLInlineList(inline) {
					if currentModelMethodsKey == "" || method == "" {
						continue
					}
					approvedModelMethodSet[currentModelMethodsKey+"."+method] = struct{}{}
				}
				continue
			}
			if indent >= specIndent+6 && strings.HasPrefix(trimmed, "- ") && currentModelMethodsKey != "" {
				method := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if method != "" {
					approvedModelMethodSet[currentModelMethodsKey+"."+method] = struct{}{}
				}
			}
		case "required_steps":
			if strings.HasPrefix(trimmed, "- name:") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))
				if step != "" {
					requiredStepSet[step] = struct{}{}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "name:") && indent >= specIndent+4 {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")))
				if step != "" {
					requiredStepSet[step] = struct{}{}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "- ") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if step != "" {
					requiredStepSet[step] = struct{}{}
				}
			}
		case "forbidden_step_names":
			if strings.HasPrefix(trimmed, "- ") {
				step := parseYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if step != "" {
					forbiddenStepSet[step] = struct{}{}
				}
			}
		}
	}

	policy.ApprovedModels = sortedStringSet(approvedModelSet)
	policy.ApprovedModelMethods = sortedStringSet(approvedModelMethodSet)
	policy.RequiredSteps = sortedStringSet(requiredStepSet)
	policy.ForbiddenStepNames = sortedStringSet(forbiddenStepSet)
	return policy, nil
}
