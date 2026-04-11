package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type workloadDocument struct {
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Containers map[string]workloadContainer `yaml:"containers"`
	Service    struct {
		Ports map[string]servicePort `yaml:"ports"`
	} `yaml:"service"`
	Resources map[string]workloadResource `yaml:"resources"`
}

type workloadContainer struct {
	Image     string            `yaml:"image"`
	Variables map[string]string `yaml:"variables"`
}

type workloadResource struct {
	Type string `yaml:"type"`
}

type servicePort struct {
	Port       int `yaml:"port"`
	TargetPort int `yaml:"targetPort"`
}

type runtimeManifests struct {
	Image     string
	Namespace string
	Content   []byte
}

var resourcePattern = regexp.MustCompile(`^\$\{resources\.([^.]+)\.(host|port)\}$`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	scorePath := flag.String("score", "", "Path to base score.yaml")
	overlayPath := flag.String("overlay", "", "Path to overlay YAML")
	namespace := flag.String("namespace", "", "Runtime namespace")
	outputPath := flag.String("output", "", "Write rendered YAML to this path instead of stdout")
	printImage := flag.Bool("print-image", false, "Print the resolved image tag only")
	flag.Parse()

	if strings.TrimSpace(*scorePath) == "" {
		return errors.New("--score is required")
	}
	if strings.TrimSpace(*overlayPath) == "" {
		return errors.New("--overlay is required")
	}

	manifests, err := renderRuntime(*scorePath, *overlayPath, *namespace)
	if err != nil {
		return err
	}

	if *printImage {
		fmt.Println(manifests.Image)
		return nil
	}

	if strings.TrimSpace(*outputPath) == "" {
		_, err = os.Stdout.Write(manifests.Content)
		return err
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	return os.WriteFile(*outputPath, manifests.Content, 0o644)
}

func renderRuntime(scorePath, overlayPath, namespace string) (runtimeManifests, error) {
	workload, err := loadMergedWorkload(scorePath, overlayPath)
	if err != nil {
		return runtimeManifests{}, err
	}

	resolvedNamespace := strings.TrimSpace(namespace)
	if resolvedNamespace == "" {
		resolvedNamespace = strings.TrimSpace(workload.Metadata.Name)
	}
	if resolvedNamespace == "" {
		return runtimeManifests{}, errors.New("metadata.name or --namespace is required")
	}

	containerName, containerSpec, err := primaryContainer(workload.Containers)
	if err != nil {
		return runtimeManifests{}, err
	}
	if strings.TrimSpace(containerSpec.Image) == "" {
		return runtimeManifests{}, errors.New("resolved Score runtime image is empty")
	}

	portName, portSpec, err := primaryServicePort(workload.Service.Ports)
	if err != nil {
		return runtimeManifests{}, err
	}

	env := envVars(resolveVariables(containerSpec.Variables, workload.Resources))
	content, err := marshalManifestSet(manifestSetInput{
		Name:          workload.Metadata.Name,
		Namespace:     resolvedNamespace,
		ContainerName: containerName,
		Image:         containerSpec.Image,
		ServiceName:   portName,
		ServicePort:   portSpec.Port,
		TargetPort:    portSpec.targetPort(),
		Env:           env,
	})
	if err != nil {
		return runtimeManifests{}, err
	}

	return runtimeManifests{
		Image:     containerSpec.Image,
		Namespace: resolvedNamespace,
		Content:   content,
	}, nil
}

func loadMergedWorkload(scorePath, overlayPath string) (workloadDocument, error) {
	baseData, err := os.ReadFile(scorePath)
	if err != nil {
		return workloadDocument{}, fmt.Errorf("read base score file: %w", err)
	}
	overlayData, err := os.ReadFile(overlayPath)
	if err != nil {
		return workloadDocument{}, fmt.Errorf("read overlay file: %w", err)
	}

	var base workloadDocument
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return workloadDocument{}, fmt.Errorf("parse base score file: %w", err)
	}
	var overlay workloadDocument
	if err := yaml.Unmarshal(overlayData, &overlay); err != nil {
		return workloadDocument{}, fmt.Errorf("parse overlay file: %w", err)
	}

	mergeWorkload(&base, overlay)
	return base, nil
}

func mergeWorkload(base *workloadDocument, overlay workloadDocument) {
	if strings.TrimSpace(overlay.Metadata.Name) != "" {
		base.Metadata.Name = overlay.Metadata.Name
	}

	if base.Containers == nil {
		base.Containers = map[string]workloadContainer{}
	}
	for name, overlayContainer := range overlay.Containers {
		current := base.Containers[name]
		if strings.TrimSpace(overlayContainer.Image) != "" {
			current.Image = overlayContainer.Image
		}
		current.Variables = mergeStringMap(current.Variables, overlayContainer.Variables)
		base.Containers[name] = current
	}

	if base.Service.Ports == nil {
		base.Service.Ports = map[string]servicePort{}
	}
	for name, overlayPort := range overlay.Service.Ports {
		current := base.Service.Ports[name]
		if overlayPort.Port != 0 {
			current.Port = overlayPort.Port
		}
		if overlayPort.TargetPort != 0 {
			current.TargetPort = overlayPort.TargetPort
		}
		base.Service.Ports[name] = current
	}

	if base.Resources == nil {
		base.Resources = map[string]workloadResource{}
	}
	for name, overlayResource := range overlay.Resources {
		current := base.Resources[name]
		if strings.TrimSpace(overlayResource.Type) != "" {
			current.Type = overlayResource.Type
		}
		base.Resources[name] = current
	}
}

func mergeStringMap(base, overlay map[string]string) map[string]string {
	if base == nil && overlay == nil {
		return nil
	}
	merged := make(map[string]string, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func primaryContainer(containers map[string]workloadContainer) (string, workloadContainer, error) {
	if len(containers) == 0 {
		return "", workloadContainer{}, errors.New("no containers declared in Score workload")
	}
	names := make([]string, 0, len(containers))
	for name := range containers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name == "main" {
			return name, containers[name], nil
		}
	}
	name := names[0]
	return name, containers[name], nil
}

func primaryServicePort(ports map[string]servicePort) (string, servicePort, error) {
	if len(ports) == 0 {
		return "", servicePort{}, errors.New("no service ports declared in Score workload")
	}
	names := make([]string, 0, len(ports))
	for name := range ports {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name == "web" {
			return name, ports[name], nil
		}
	}
	name := names[0]
	return name, ports[name], nil
}

func resolveVariables(variables map[string]string, resources map[string]workloadResource) map[string]string {
	if len(variables) == 0 {
		return nil
	}
	resolved := make(map[string]string, len(variables))
	for key, value := range variables {
		matches := resourcePattern.FindStringSubmatch(strings.TrimSpace(value))
		if len(matches) != 3 {
			resolved[key] = value
			continue
		}

		resourceName := matches[1]
		field := matches[2]
		resourceSpec := resources[resourceName]
		resolved[key] = resourceValue(resourceName, resourceSpec.Type, field, value)
	}
	return resolved
}

func resourceValue(resourceName, resourceType, field, fallback string) string {
	resourceType = strings.TrimSpace(resourceType)
	switch field {
	case "host":
		switch resourceType {
		case "postgres":
			return "postgres.platform.svc.cluster.local"
		case "redis":
			return "redis.platform.svc.cluster.local"
		case "dns":
			return "dns.platform.svc.cluster.local"
		}
		return fmt.Sprintf("%s.platform.svc.cluster.local", resourceName)
	case "port":
		switch resourceType {
		case "postgres":
			return "5432"
		case "redis":
			return "6379"
		}
	}
	return fallback
}

type envVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func envVars(values map[string]string) []envVar {
	if len(values) == 0 {
		return nil
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]envVar, 0, len(names)+1)
	for _, name := range names {
		out = append(out, envVar{Name: name, Value: values[name]})
	}
	return out
}

type manifestSetInput struct {
	Name          string
	Namespace     string
	ContainerName string
	Image         string
	ServiceName   string
	ServicePort   int
	TargetPort    int
	Env           []envVar
}

func marshalManifestSet(input manifestSetInput) ([]byte, error) {
	labels := map[string]string{
		"app.kubernetes.io/name": input.Name,
		"app.kubernetes.io/part-of": "scoredev-paas",
	}

	objects := []any{
		map[string]any{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]any{
				"name": input.Namespace,
			},
		},
		map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]any{
				"name":      input.Name,
				"namespace": input.Namespace,
				"labels":    labels,
			},
			"spec": map[string]any{
				"replicas": 1,
				"selector": map[string]any{
					"matchLabels": labels,
				},
				"template": map[string]any{
					"metadata": map[string]any{
						"labels": labels,
					},
					"spec": map[string]any{
						"containers": []any{
							map[string]any{
								"name":            input.ContainerName,
								"image":           input.Image,
								"imagePullPolicy": "IfNotPresent",
								"ports": []any{
									map[string]any{
										"containerPort": input.TargetPort,
										"name":          input.ServiceName,
									},
								},
								"env": input.Env,
								"readinessProbe": map[string]any{
									"httpGet": map[string]any{
										"path": "/healthz",
										"port": input.TargetPort,
									},
									"initialDelaySeconds": 2,
									"periodSeconds":       3,
								},
								"livenessProbe": map[string]any{
									"httpGet": map[string]any{
										"path": "/healthz",
										"port": input.TargetPort,
									},
									"initialDelaySeconds": 5,
									"periodSeconds":       5,
								},
							},
						},
					},
				},
			},
		},
		map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]any{
				"name":      input.Name,
				"namespace": input.Namespace,
				"labels":    labels,
			},
			"spec": map[string]any{
				"selector": labels,
				"ports": []any{
					map[string]any{
						"name":       input.ServiceName,
						"port":       input.ServicePort,
						"targetPort": input.TargetPort,
					},
				},
			},
		},
	}

	var out strings.Builder
	for i, object := range objects {
		rendered, err := yaml.Marshal(object)
		if err != nil {
			return nil, fmt.Errorf("marshal manifest %d: %w", i, err)
		}
		if i > 0 {
			out.WriteString("---\n")
		}
		out.Write(rendered)
	}
	return []byte(out.String()), nil
}

func (p servicePort) targetPort() int {
	if p.TargetPort != 0 {
		return p.TargetPort
	}
	return p.Port
}
