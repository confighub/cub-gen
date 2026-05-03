package springboot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/confighub/cub-gen/internal/detect"
	"github.com/confighub/cub-gen/internal/model"
)

// InitOptions configures the springboot init command.
type InitOptions struct {
	SourcePath string
	OutputPath string
	AppName    string
	Namespace  string
	DryRun     bool
}

// InitResult captures what was generated.
type InitResult struct {
	AppName      string    `json:"app_name"`
	SourcePath   string    `json:"source_path"`
	OutputPath   string    `json:"output_path"`
	FilesCreated []string  `json:"files_created"`
	Detection    Detection `json:"detection"`
	Checklist    []string  `json:"checklist"`
}

// Detection holds Spring Boot detection info.
type Detection struct {
	Kind       string   `json:"kind"`
	Profile    string   `json:"profile"`
	Inputs     []string `json:"inputs"`
	Confidence float64  `json:"confidence"`
}

// Init generates starter cub-gen material for a Spring Boot app.
func Init(opts InitOptions) (*InitResult, error) {
	if opts.SourcePath == "" {
		return nil, fmt.Errorf("source path is required")
	}

	absSource, err := filepath.Abs(opts.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve source path: %w", err)
	}

	st, err := os.Stat(absSource)
	if err != nil {
		return nil, fmt.Errorf("stat source: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("source path is not a directory: %s", absSource)
	}

	// Detect Spring Boot app
	detectionResult, err := detect.ScanRepo(absSource, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("scan repo: %w", err)
	}

	var springDetection *model.GeneratorDetection
	for i := range detectionResult.Generators {
		if detectionResult.Generators[i].Kind == model.GeneratorSpringBoot {
			springDetection = &detectionResult.Generators[i]
			break
		}
	}

	if springDetection == nil {
		return nil, fmt.Errorf("no Spring Boot app detected in %s\n\nExpected: pom.xml (or build.gradle) + src/main/resources/application.yaml", absSource)
	}

	// Derive app name
	appName := opts.AppName
	if appName == "" {
		appName = springDetection.Name
	}
	if appName == "" {
		appName = filepath.Base(absSource)
	}

	// Derive output path
	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = absSource
	}
	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, fmt.Errorf("resolve output path: %w", err)
	}

	namespace := opts.Namespace
	if namespace == "" {
		namespace = "apps"
	}

	result := &InitResult{
		AppName:    appName,
		SourcePath: absSource,
		OutputPath: absOutput,
		Detection: Detection{
			Kind:       string(springDetection.Kind),
			Profile:    springDetection.Profile,
			Inputs:     springDetection.Inputs,
			Confidence: springDetection.Confidence,
		},
	}

	if opts.DryRun {
		result.FilesCreated = dryRunFiles(appName)
		result.Checklist = generateChecklist(appName)
		return result, nil
	}

	// Create directories
	dirs := []string{
		filepath.Join(absOutput, "platform", "base"),
		filepath.Join(absOutput, "platform", "overlays", "prod"),
		filepath.Join(absOutput, "operational"),
		filepath.Join(absOutput, "confighub"),
		filepath.Join(absOutput, ".cub-gen"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Generate files
	files := map[string]string{
		filepath.Join(absOutput, "platform", "base", "runtime-policy.yaml"):         runtimePolicyContent(appName),
		filepath.Join(absOutput, "platform", "overlays", "prod", "slo-policy.yaml"): sloPolicyContent(appName),
		filepath.Join(absOutput, "operational", "field-routes.yaml"):                fieldRoutesContent(appName),
		filepath.Join(absOutput, "confighub", appName+"-dev.yaml"):                  confighubUnitContent(appName, "dev", namespace),
		filepath.Join(absOutput, "confighub", appName+"-stage.yaml"):                confighubUnitContent(appName, "stage", namespace),
		filepath.Join(absOutput, "confighub", appName+"-prod.yaml"):                 confighubUnitContent(appName, "prod", namespace),
		filepath.Join(absOutput, ".cub-gen", "config.yaml"):                         cubGenConfigContent(appName),
	}

	for path, content := range files {
		// Don't overwrite existing files
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("write %s: %w", path, err)
		}
		relPath, _ := filepath.Rel(absOutput, path)
		result.FilesCreated = append(result.FilesCreated, relPath)
	}

	result.Checklist = generateChecklist(appName)
	return result, nil
}

func dryRunFiles(appName string) []string {
	return []string{
		"platform/base/runtime-policy.yaml",
		"platform/overlays/prod/slo-policy.yaml",
		"operational/field-routes.yaml",
		"confighub/" + appName + "-dev.yaml",
		"confighub/" + appName + "-stage.yaml",
		"confighub/" + appName + "-prod.yaml",
		".cub-gen/config.yaml",
	}
}

func generateChecklist(appName string) []string {
	return []string{
		"Review platform/base/runtime-policy.yaml and set your platform owner",
		"Review operational/field-routes.yaml and customize ownership rules",
		"Review confighub/*.yaml and set correct image, ports, and environment",
		"Add actual Kubernetes manifests to operational/ (deployment.yaml, service.yaml, configmap.yaml)",
		"Run: cub-gen gitops discover ./" + appName,
		"Run: cub-gen gitops import ./" + appName + " ./" + appName,
	}
}

func runtimePolicyContent(appName string) string {
	return fmt.Sprintf(`# Platform runtime policy for %s
# This is a starter skeleton. Customize for your platform.

apiVersion: platform.confighub.io/v1alpha1
kind: RuntimePolicy
metadata:
  name: %s-defaults
spec:
  owner: platform-engineering
  requireActuatorHealth: true
  # managedDatasource: postgres-shared  # uncomment if using managed datasource
`, appName, appName)
}

func sloPolicyContent(appName string) string {
	return fmt.Sprintf(`# SLO policy for %s production
# This is a starter skeleton. Customize for your SLOs.

apiVersion: platform.confighub.io/v1alpha1
kind: SLOPolicy
metadata:
  name: %s-prod-slo
spec:
  availability: 99.9
  latencyP99Ms: 500
`, appName, appName)
}

func fieldRoutesContent(appName string) string {
	appPrefix := strings.ReplaceAll(appName, "-", "")
	return fmt.Sprintf(`# Field ownership routes for %s
# These defaults match the springboot-paas example pattern.
# Customize based on your app's actual field ownership needs.

routes:
  # App-owned feature flags: mutate directly in ConfigHub
  - match: feature.%s.*
    owner: app-team
    defaultAction: mutable-in-ch
    reason: Per-deployment feature tuning is safe to keep in ConfigHub.
    sourcePath: src/main/resources/application-prod.yaml
    sourceField: feature.%s.*

  # App-owned cache config: requires source change
  - match: spring.cache.*
    owner: app-team
    defaultAction: lift-upstream
    reason: Cache adoption changes the app contract and should update upstream app inputs.
    sourcePath: src/main/resources/application.yaml
    sourceField: spring.cache.*
    proposalFiles:
      - pom.xml
      - src/main/resources/application.yaml

  # Platform-owned datasource: blocked from app changes
  - match: spring.datasource.*
    owner: platform-engineering
    defaultAction: generator-owned
    reason: Datasource connectivity is part of the managed platform boundary.
    sourcePath: platform/base/runtime-policy.yaml
    sourceField: spring.datasource.*

  # Platform-owned security: blocked from app changes
  - match: securityContext.*
    owner: platform-engineering
    defaultAction: generator-owned
    reason: Runtime hardening must remain platform-controlled.
    sourcePath: platform/base/runtime-policy.yaml
    sourceField: securityContext.*
`, appName, appPrefix, appPrefix)
}

func confighubUnitContent(appName, env, namespace string) string {
	image := fmt.Sprintf("ghcr.io/example/%s:1.0.0", appName)
	datasourceURL := fmt.Sprintf("jdbc:postgresql://postgres.platform.svc:5432/%s", strings.ReplaceAll(appName, "-", "_"))

	port := "8080"
	replicas := "1"
	if env == "prod" {
		replicas = "2"
	}

	return fmt.Sprintf(`# ConfigHub unit for %s (%s environment)
# This is a starter skeleton. Customize for your app.

apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
data:
  application.yaml: |
    spring:
      application:
        name: %s
      datasource:
        url: %s
        username: %s
    management:
      endpoints:
        web:
          exposure:
            include: health,info
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
spec:
  replicas: %s
  selector:
    matchLabels:
      app.kubernetes.io/name: %s
  template:
    metadata:
      labels:
        app.kubernetes.io/name: %s
    spec:
      serviceAccountName: %s
      containers:
        - name: %s
          image: %s
          ports:
            - containerPort: %s
          env:
            - name: SPRING_CONFIG_ADDITIONAL_LOCATION
              value: /config/
            - name: SPRING_PROFILES_ACTIVE
              value: %s
          volumeMounts:
            - name: app-config
              mountPath: /config
      volumes:
        - name: app-config
          configMap:
            name: %s-config
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app.kubernetes.io/name: %s
  ports:
    - name: http
      port: 80
      targetPort: %s
`,
		appName, env,
		appName, namespace, appName,
		appName, namespace, appName,
		appName, datasourceURL, strings.ReplaceAll(appName, "-", "_"),
		appName, namespace, appName, replicas, appName, appName, appName, appName, image, port, env,
		appName,
		appName, namespace, appName, port,
	)
}

func cubGenConfigContent(appName string) string {
	return fmt.Sprintf(`# cub-gen configuration for %s
# Generated by: cub-gen springboot init

generator:
  kind: springboot
  profile: springboot-paas

app:
  name: %s

# Uncomment to customize paths
# paths:
#   platform: platform/
#   operational: operational/
#   confighub: confighub/
`, appName, appName)
}
