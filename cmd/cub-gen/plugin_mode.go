package main

import (
	"os"
	"strings"
)

const (
	envCubPlugin = "CUB_PLUGIN"
	envCubServer = "CUB_SERVER"
	envCubToken  = "CUB_TOKEN"
)

func isPluginMode() bool {
	return strings.TrimSpace(os.Getenv(envCubPlugin)) != ""
}

func pluginHelpLine(line string) string {
	if !isPluginMode() {
		return line
	}

	replacer := strings.NewReplacer(
		"cub-gen:", "cub gen:",
		"cub-gen ", "cub gen ",
		"cub-gen.", "cub gen.",
		"cub-gen,", "cub gen,",
		"`cub-gen", "`cub gen",
		"'cub-gen", "'cub gen",
		"\"cub-gen", "\"cub gen",
		"| cub-gen", "| cub gen",
	)
	return replacer.Replace(line)
}

func normalizePluginArgs(args []string) []string {
	if !isPluginMode() || len(args) == 0 {
		return args
	}

	switch args[0] {
	case "discover":
		return prependPluginCommand("gitops", args)
	case "import", "render":
		return append([]string{"gitops", "import"}, args[1:]...)
	case "cleanup":
		return prependPluginCommand("gitops", args)
	case "fanout":
		return prependPluginCommand("platform", args)
	case "adapt":
		return prependPluginCommand("platform", args)
	case "preview", "run", "diff", "revision-diff", "impact", "explain":
		return prependPluginCommand("change", args)
	case "ingest", "link", "decision", "promote":
		return prependPluginCommand("bridge", args)
	case "bundle":
		return normalizePluginBundleArgs(args)
	default:
		return args
	}
}

func prependPluginCommand(parent string, args []string) []string {
	out := make([]string, 0, len(args)+1)
	out = append(out, parent)
	out = append(out, args...)
	return out
}

func normalizePluginBundleArgs(args []string) []string {
	if len(args) == 1 {
		return []string{"publish"}
	}
	switch args[1] {
	case "help":
		return []string{"publish", "--help"}
	case "-h", "--help":
		return append([]string{"publish"}, args[1:]...)
	case "publish":
		return normalizePluginBundleSubcommandArgs("publish", args[2:])
	case "verify":
		return normalizePluginBundleSubcommandArgs("verify", args[2:])
	case "attest":
		return normalizePluginBundleSubcommandArgs("attest", args[2:])
	case "verify-attestation":
		return normalizePluginBundleSubcommandArgs("verify-attestation", args[2:])
	default:
		return append([]string{"publish"}, args[1:]...)
	}
}

func normalizePluginBundleSubcommandArgs(command string, args []string) []string {
	if len(args) == 1 && args[0] == "help" {
		return []string{command, "--help"}
	}
	return append([]string{command}, args...)
}

func defaultConfigHubBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("CONFIGHUB_BASE_URL")); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(envCubServer))
}

func defaultConfigHubToken() string {
	if value := strings.TrimSpace(os.Getenv("CONFIGHUB_TOKEN")); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(envCubToken))
}
