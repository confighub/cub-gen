package importer

import (
	"sort"
	"strings"
)

func parseYAMLScalar(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return strings.TrimSpace(value)
}

func parseYAMLInlineList(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil
	}
	body := strings.TrimSpace(value[1 : len(value)-1])
	if body == "" {
		return nil
	}
	parts := strings.Split(body, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := parseYAMLScalar(part)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return uniqueSortedStrings(items)
}

func sortedStringSet(in map[string]struct{}) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for value := range in {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return sortedStringSet(set)
}

func differenceStrings(values, baseline []string) []string {
	if len(values) == 0 {
		return nil
	}
	baselineSet := map[string]struct{}{}
	for _, v := range baseline {
		baselineSet[strings.TrimSpace(v)] = struct{}{}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := baselineSet[key]; ok {
			continue
		}
		out = append(out, key)
	}
	return uniqueSortedStrings(out)
}

func intersectionStrings(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	setB := map[string]struct{}{}
	for _, value := range b {
		setB[strings.TrimSpace(value)] = struct{}{}
	}
	out := make([]string, 0, len(a))
	for _, value := range a {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := setB[key]; ok {
			out = append(out, key)
		}
	}
	return uniqueSortedStrings(out)
}

func differenceAgainstAllowList(observed, allowList []string) []string {
	if len(observed) == 0 || len(allowList) == 0 {
		return nil
	}
	allowedSet := map[string]struct{}{}
	for _, value := range allowList {
		allowedSet[strings.TrimSpace(value)] = struct{}{}
	}
	out := make([]string, 0, len(observed))
	for _, value := range observed {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := allowedSet[key]; ok {
			continue
		}
		out = append(out, key)
	}
	return uniqueSortedStrings(out)
}
