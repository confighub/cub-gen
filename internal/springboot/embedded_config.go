package springboot

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type SetEmbeddedConfigOptions struct {
	FilePath        string
	ConfigMapName   string
	ConfigKey       string
	FieldRoutesPath string
	FieldPath       string
	Value           string
}

type SetEmbeddedConfigResult struct {
	FilePath      string      `json:"file_path"`
	ConfigMapName string      `json:"configmap_name"`
	ConfigKey     string      `json:"config_key"`
	FieldPath     string      `json:"field_path"`
	Allowed       bool        `json:"allowed"`
	Action        RouteAction `json:"action,omitempty"`
	Owner         string      `json:"owner,omitempty"`
	Reason        string      `json:"reason,omitempty"`
	MatchedRule   string      `json:"matched_rule,omitempty"`
	OldValue      string      `json:"old_value,omitempty"`
	NewValue      string      `json:"new_value,omitempty"`
	Updated       bool        `json:"updated"`
}

func SetEmbeddedConfig(opts SetEmbeddedConfigOptions) (*SetEmbeddedConfigResult, error) {
	if strings.TrimSpace(opts.FilePath) == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	if strings.TrimSpace(opts.FieldPath) == "" {
		return nil, fmt.Errorf("field_path is required")
	}
	if strings.TrimSpace(opts.Value) == "" {
		return nil, fmt.Errorf("value is required")
	}

	filePath, err := filepath.Abs(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("resolve file path: %w", err)
	}

	configKey := strings.TrimSpace(opts.ConfigKey)
	if configKey == "" {
		configKey = "application.yaml"
	}

	result := &SetEmbeddedConfigResult{
		FilePath:      filePath,
		ConfigMapName: strings.TrimSpace(opts.ConfigMapName),
		ConfigKey:     configKey,
		FieldPath:     strings.TrimSpace(opts.FieldPath),
		Allowed:       true,
	}

	if strings.TrimSpace(opts.FieldRoutesPath) != "" {
		validation, err := ValidateMutation(ValidateMutationOptions{
			FieldRoutesPath: opts.FieldRoutesPath,
			FieldPath:       result.FieldPath,
		})
		if err != nil {
			return nil, err
		}
		result.Allowed = validation.Allowed
		result.Action = validation.Action
		result.Owner = validation.Owner
		result.Reason = validation.Reason
		result.MatchedRule = validation.MatchedRule
		if !validation.Allowed {
			return result, nil
		}
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read config payload: %w", err)
	}

	docs, err := decodeDocuments(raw)
	if err != nil {
		return nil, fmt.Errorf("parse config payload: %w", err)
	}

	configMapRoot, configMapName, err := findConfigMapWithKey(docs, result.ConfigMapName, configKey)
	if err != nil {
		return nil, err
	}
	result.ConfigMapName = configMapName

	dataNode := mappingValue(configMapRoot, "data")
	configValueNode := mappingValue(dataNode, configKey)
	if configValueNode == nil {
		return nil, fmt.Errorf("config map %s does not contain data[%q]", configMapName, configKey)
	}

	embeddedDoc, err := decodeSingleDocument([]byte(configValueNode.Value))
	if err != nil {
		return nil, fmt.Errorf("parse embedded config %q: %w", configKey, err)
	}
	if len(embeddedDoc.Content) == 0 {
		return nil, fmt.Errorf("embedded config %q is empty", configKey)
	}

	targetNode, err := lookupPath(embeddedDoc.Content[0], strings.Split(result.FieldPath, "."))
	if err != nil {
		return nil, fmt.Errorf("lookup field %s in %s: %w", result.FieldPath, configKey, err)
	}
	if targetNode.Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("field %s in %s is not a scalar value", result.FieldPath, configKey)
	}

	newValueNode, err := decodeScalar(opts.Value)
	if err != nil {
		return nil, fmt.Errorf("parse value: %w", err)
	}

	result.OldValue = scalarString(targetNode)
	result.NewValue = scalarString(newValueNode)
	result.Updated = result.OldValue != result.NewValue
	*targetNode = *cloneNode(newValueNode)

	embeddedBytes, err := yaml.Marshal(embeddedDoc.Content[0])
	if err != nil {
		return nil, fmt.Errorf("encode embedded config %q: %w", configKey, err)
	}
	configValueNode.Kind = yaml.ScalarNode
	configValueNode.Tag = "!!str"
	configValueNode.Style = yaml.LiteralStyle
	configValueNode.Value = strings.TrimSuffix(string(embeddedBytes), "\n")

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	for _, doc := range docs {
		if err := enc.Encode(doc); err != nil {
			_ = enc.Close()
			return nil, fmt.Errorf("encode config payload: %w", err)
		}
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("finalize config payload: %w", err)
	}

	if err := os.WriteFile(filePath, out.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("write config payload: %w", err)
	}

	return result, nil
}

func decodeDocuments(raw []byte) ([]*yaml.Node, error) {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	docs := make([]*yaml.Node, 0, 4)
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if len(doc.Content) == 0 {
			continue
		}
		docs = append(docs, &doc)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no YAML documents found")
	}
	return docs, nil
}

func decodeSingleDocument(raw []byte) (*yaml.Node, error) {
	docs, err := decodeDocuments(raw)
	if err != nil {
		return nil, err
	}
	if len(docs) != 1 {
		return nil, fmt.Errorf("expected exactly one YAML document, found %d", len(docs))
	}
	return docs[0], nil
}

func findConfigMapWithKey(docs []*yaml.Node, configMapName, configKey string) (*yaml.Node, string, error) {
	matches := make([]*yaml.Node, 0, 1)
	names := make([]string, 0, 1)
	for _, doc := range docs {
		if len(doc.Content) == 0 {
			continue
		}
		root := doc.Content[0]
		kindNode := mappingValue(root, "kind")
		if kindNode == nil || kindNode.Value != "ConfigMap" {
			continue
		}
		name := configMapMetadataName(root)
		if strings.TrimSpace(configMapName) != "" && name != strings.TrimSpace(configMapName) {
			continue
		}
		dataNode := mappingValue(root, "data")
		if dataNode == nil || mappingValue(dataNode, configKey) == nil {
			continue
		}
		matches = append(matches, root)
		names = append(names, name)
	}
	if len(matches) == 0 {
		if strings.TrimSpace(configMapName) != "" {
			return nil, "", fmt.Errorf("no ConfigMap named %q with data[%q] found", configMapName, configKey)
		}
		return nil, "", fmt.Errorf("no ConfigMap with data[%q] found; use --configmap if the payload contains multiple ConfigMaps", configKey)
	}
	if len(matches) > 1 && strings.TrimSpace(configMapName) == "" {
		return nil, "", fmt.Errorf("multiple ConfigMaps with data[%q] found; use --configmap to choose one", configKey)
	}
	return matches[0], names[0], nil
}

func configMapMetadataName(root *yaml.Node) string {
	metadataNode := mappingValue(root, "metadata")
	if metadataNode == nil {
		return ""
	}
	nameNode := mappingValue(metadataNode, "name")
	if nameNode == nil {
		return ""
	}
	return strings.TrimSpace(nameNode.Value)
}

func mappingValue(root *yaml.Node, key string) *yaml.Node {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

func lookupPath(root *yaml.Node, parts []string) (*yaml.Node, error) {
	current := root
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty path segment")
		}
		if current.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("path segment %q is not inside a mapping", part)
		}
		next := mappingValue(current, part)
		if next == nil {
			return nil, fmt.Errorf("missing path segment %q", part)
		}
		current = next
	}
	return current, nil
}

func decodeScalar(raw string) (*yaml.Node, error) {
	doc, err := decodeSingleDocument([]byte(raw))
	if err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.ScalarNode {
		return nil, fmt.Errorf("value must be a YAML scalar")
	}
	return doc.Content[0], nil
}

func scalarString(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Value)
}

func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	cloned := *node
	if len(node.Content) > 0 {
		cloned.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			cloned.Content[i] = cloneNode(child)
		}
	}
	return &cloned
}
