package springboot

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RouteAction defines what happens when a field matches a route.
type RouteAction string

const (
	ActionMutableInCH    RouteAction = "mutable-in-ch"
	ActionLiftUpstream   RouteAction = "lift-upstream"
	ActionGeneratorOwned RouteAction = "generator-owned"
)

// FieldRoute represents a single field ownership rule.
type FieldRoute struct {
	Match         string      `yaml:"match"`
	Owner         string      `yaml:"owner"`
	DefaultAction RouteAction `yaml:"defaultAction"`
	Reason        string      `yaml:"reason"`
}

// FieldRoutes is the root structure of field-routes.yaml.
type FieldRoutes struct {
	Routes []FieldRoute `yaml:"routes"`
}

// ValidationResult captures the outcome of a mutation validation.
type ValidationResult struct {
	FieldPath   string      `json:"field_path"`
	Allowed     bool        `json:"allowed"`
	Action      RouteAction `json:"action"`
	Owner       string      `json:"owner"`
	Reason      string      `json:"reason"`
	MatchedRule string      `json:"matched_rule,omitempty"`
}

// ValidateMutationOptions configures mutation validation.
type ValidateMutationOptions struct {
	FieldRoutesPath string
	FieldPath       string
}

// ValidateMutation checks if a proposed mutation to a field path is allowed.
func ValidateMutation(opts ValidateMutationOptions) (*ValidationResult, error) {
	if opts.FieldRoutesPath == "" {
		return nil, fmt.Errorf("field_routes_path is required")
	}
	if opts.FieldPath == "" {
		return nil, fmt.Errorf("field_path is required")
	}

	absPath, err := filepath.Abs(opts.FieldRoutesPath)
	if err != nil {
		return nil, fmt.Errorf("resolve field routes path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read field routes: %w", err)
	}

	var routes FieldRoutes
	if err := yaml.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("parse field routes: %w", err)
	}

	fieldPath := strings.TrimSpace(opts.FieldPath)

	// Find the first matching route
	for _, route := range routes.Routes {
		if matchRoute(route.Match, fieldPath) {
			allowed := route.DefaultAction == ActionMutableInCH
			return &ValidationResult{
				FieldPath:   fieldPath,
				Allowed:     allowed,
				Action:      route.DefaultAction,
				Owner:       route.Owner,
				Reason:      route.Reason,
				MatchedRule: route.Match,
			}, nil
		}
	}

	// No matching rule - default to allowed (app can mutate unknown fields)
	return &ValidationResult{
		FieldPath: fieldPath,
		Allowed:   true,
		Action:    ActionMutableInCH,
		Owner:     "app-team",
		Reason:    "No matching field route rule; defaults to app-team ownership",
	}, nil
}

// matchRoute checks if a field path matches a route pattern.
// Patterns use glob-style wildcards: * matches any characters (including dots).
func matchRoute(pattern, fieldPath string) bool {
	matched, err := path.Match(strings.TrimSpace(pattern), strings.TrimSpace(fieldPath))
	if err != nil {
		return false
	}
	return matched
}

// EnforceMutation validates a mutation and returns an error if blocked.
// This is the enforcement entry point.
func EnforceMutation(opts ValidateMutationOptions) error {
	result, err := ValidateMutation(opts)
	if err != nil {
		return err
	}

	if !result.Allowed {
		return &MutationBlockedError{
			FieldPath: result.FieldPath,
			Owner:     result.Owner,
			Action:    result.Action,
			Reason:    result.Reason,
			Rule:      result.MatchedRule,
		}
	}

	return nil
}

// MutationBlockedError indicates a mutation was blocked by field routes.
type MutationBlockedError struct {
	FieldPath string
	Owner     string
	Action    RouteAction
	Reason    string
	Rule      string
}

func (e *MutationBlockedError) Error() string {
	return fmt.Sprintf("mutation blocked: field %q is %s-owned (action=%s, rule=%s): %s",
		e.FieldPath, e.Owner, e.Action, e.Rule, e.Reason)
}

// IsMutationBlocked returns true if the error is a MutationBlockedError.
func IsMutationBlocked(err error) bool {
	var blocked *MutationBlockedError
	return errors.As(err, &blocked)
}
