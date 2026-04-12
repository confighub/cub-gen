package main

import (
	"flag"
	"strings"

	"github.com/confighub/cub-gen/internal/importer"
	"github.com/confighub/cub-gen/internal/model"
)

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	if m == nil {
		return ""
	}
	return strings.Join(*m, ",")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

type helmCLIOverrideFlagSet struct {
	set       multiStringFlag
	setString multiStringFlag
	setFile   multiStringFlag
}

func addHelmCLIOverrideFlags(fs *flag.FlagSet) *helmCLIOverrideFlagSet {
	flags := &helmCLIOverrideFlagSet{}
	fs.Var(&flags.set, "set", "Helm-style override (key=value); repeat or comma-separate entries")
	fs.Var(&flags.setString, "set-string", "Helm-style string override (key=value); repeat or comma-separate entries")
	fs.Var(&flags.setFile, "set-file", "Helm-style file override (key=path); repeat or comma-separate entries")
	return flags
}

func (f *helmCLIOverrideFlagSet) parse() ([]model.HelmCLIOverride, error) {
	if f == nil {
		return nil, nil
	}
	return importer.ParseHelmCLIOverrides(f.set, f.setString, f.setFile)
}
