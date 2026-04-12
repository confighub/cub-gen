package importer

import (
	"fmt"
	"strings"

	"github.com/confighub/cub-gen/internal/model"
)

type ImportOptions struct {
	HelmCLIOverrides []model.HelmCLIOverride
}

func ParseHelmCLIOverrides(setValues, setStringValues, setFileValues []string) ([]model.HelmCLIOverride, error) {
	overrides := make([]model.HelmCLIOverride, 0, len(setValues)+len(setStringValues)+len(setFileValues))

	parsedSet, err := parseHelmCLIOverrideAssignments("set", setValues)
	if err != nil {
		return nil, err
	}
	overrides = append(overrides, parsedSet...)

	parsedSetString, err := parseHelmCLIOverrideAssignments("set-string", setStringValues)
	if err != nil {
		return nil, err
	}
	overrides = append(overrides, parsedSetString...)

	parsedSetFile, err := parseHelmCLIOverrideAssignments("set-file", setFileValues)
	if err != nil {
		return nil, err
	}
	overrides = append(overrides, parsedSetFile...)

	return overrides, nil
}

func HelmCLIOverrideDisplay(override model.HelmCLIOverride) string {
	switch override.Flag {
	case "set-file":
		return fmt.Sprintf("--%s %s=%s", override.Flag, override.Key, override.FilePath)
	default:
		return fmt.Sprintf("--%s %s=%s", override.Flag, override.Key, override.Value)
	}
}

func HelmCLIOverrideDigestParts(overrides []model.HelmCLIOverride) []string {
	parts := make([]string, 0, len(overrides))
	for _, override := range overrides {
		part := override.Flag + ":" + override.Key + "="
		if override.Flag == "set-file" {
			part += override.FilePath
		} else {
			part += override.Value
		}
		parts = append(parts, part)
	}
	return parts
}

func parseHelmCLIOverrideAssignments(flagName string, rawValues []string) ([]model.HelmCLIOverride, error) {
	overrides := make([]model.HelmCLIOverride, 0, len(rawValues))
	for _, rawValue := range rawValues {
		assignments := splitHelmCLIOverrideAssignments(rawValue)
		for _, assignment := range assignments {
			key, value, err := splitHelmCLIOverrideAssignment(assignment)
			if err != nil {
				return nil, fmt.Errorf("--%s %q: %w", flagName, rawValue, err)
			}
			override := model.HelmCLIOverride{
				Flag: flagName,
				Key:  key,
			}
			if flagName == "set-file" {
				override.FilePath = value
				if strings.TrimSpace(override.FilePath) == "" {
					return nil, fmt.Errorf("--%s %q: file path is required", flagName, rawValue)
				}
			} else {
				override.Value = value
			}
			overrides = append(overrides, override)
		}
	}
	return overrides, nil
}

func splitHelmCLIOverrideAssignments(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var (
		out     []string
		current strings.Builder
		escaped bool
	)
	for _, r := range raw {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case ',':
			out = append(out, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	out = append(out, current.String())
	return out
}

func splitHelmCLIOverrideAssignment(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("expected key=value")
	}

	var (
		keyBuilder   strings.Builder
		valueBuilder strings.Builder
		escaped      bool
		seenEquals   bool
	)
	for _, r := range trimmed {
		if escaped {
			if seenEquals {
				valueBuilder.WriteRune(r)
			} else {
				keyBuilder.WriteRune(r)
			}
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case '=':
			if !seenEquals {
				seenEquals = true
				continue
			}
			valueBuilder.WriteRune(r)
		default:
			if seenEquals {
				valueBuilder.WriteRune(r)
			} else {
				keyBuilder.WriteRune(r)
			}
		}
	}
	if escaped {
		if seenEquals {
			valueBuilder.WriteRune('\\')
		} else {
			keyBuilder.WriteRune('\\')
		}
	}
	if !seenEquals {
		return "", "", fmt.Errorf("expected key=value")
	}

	key := strings.TrimSpace(keyBuilder.String())
	if key == "" {
		return "", "", fmt.Errorf("key is required")
	}
	return key, strings.TrimSpace(valueBuilder.String()), nil
}
