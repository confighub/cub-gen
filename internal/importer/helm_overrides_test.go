package importer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/confighub/cub-gen/internal/model"
)

func TestParseHelmCLIOverrides(t *testing.T) {
	overrides, err := ParseHelmCLIOverrides(
		[]string{"image.tag=v1.2.4,appConfig.logLevel=debug"},
		[]string{"database.password=changeme"},
		[]string{"tlsCert=./certs/prod.pem"},
	)
	if err != nil {
		t.Fatalf("ParseHelmCLIOverrides returned error: %v", err)
	}

	if len(overrides) != 4 {
		t.Fatalf("expected 4 overrides, got %d", len(overrides))
	}
	if overrides[0].Flag != "set" || overrides[0].Key != "image.tag" || overrides[0].Value != "v1.2.4" {
		t.Fatalf("unexpected first override: %+v", overrides[0])
	}
	if overrides[1].Flag != "set" || overrides[1].Key != "appConfig.logLevel" || overrides[1].Value != "debug" {
		t.Fatalf("unexpected second override: %+v", overrides[1])
	}
	if overrides[2].Flag != "set-string" || overrides[2].Key != "database.password" || overrides[2].Value != "changeme" {
		t.Fatalf("unexpected set-string override: %+v", overrides[2])
	}
	if overrides[3].Flag != "set-file" || overrides[3].Key != "tlsCert" || overrides[3].FilePath != "./certs/prod.pem" {
		t.Fatalf("unexpected set-file override: %+v", overrides[3])
	}
}

func TestImportRepoWithHelmCLIOverridesAddsProvenance(t *testing.T) {
	repo := filepath.Join("..", "..", "examples", "helm-paas")

	plain, err := ImportRepo(repo, "main", "platform")
	if err != nil {
		t.Fatalf("ImportRepo returned error: %v", err)
	}

	overrideImport, err := ImportRepoWithOptions(repo, "main", "platform", ImportOptions{
		HelmCLIOverrides: []model.HelmCLIOverride{{
			Flag:  "set",
			Key:   "image.tag",
			Value: "v9.9.9",
		}},
	})
	if err != nil {
		t.Fatalf("ImportRepoWithOptions returned error: %v", err)
	}

	if plain.ChangeID == overrideImport.ChangeID {
		t.Fatalf("expected override import to change change_id, got %q", overrideImport.ChangeID)
	}

	if len(overrideImport.Provenance) != 1 {
		t.Fatalf("expected 1 provenance record, got %d", len(overrideImport.Provenance))
	}
	prov := overrideImport.Provenance[0]
	if len(prov.HelmCLIOverrides) != 1 {
		t.Fatalf("expected 1 helm_cli_override, got %+v", prov.HelmCLIOverrides)
	}
	if got := prov.HelmCLIOverrides[0]; got.Flag != "set" || got.Key != "image.tag" || got.Value != "v9.9.9" {
		t.Fatalf("unexpected helm_cli_override: %+v", got)
	}
	if len(prov.FieldOriginMap) == 0 {
		t.Fatalf("expected field origins, got none")
	}
	if prov.FieldOriginMap[0].Transform != "helm-cli-override" {
		t.Fatalf("expected override field origin first, got %+v", prov.FieldOriginMap[0])
	}
	if prov.FieldOriginMap[0].Confidence != 1.0 {
		t.Fatalf("expected override confidence 1.0, got %v", prov.FieldOriginMap[0].Confidence)
	}
	if !strings.Contains(prov.FieldOriginMap[0].SourcePath, "--set image.tag=v9.9.9") {
		t.Fatalf("expected override source path, got %q", prov.FieldOriginMap[0].SourcePath)
	}

	foundOverrideSource := false
	for _, source := range prov.Sources {
		if source.Role == "cli-override" && strings.Contains(source.Path, "--set image.tag=v9.9.9") {
			foundOverrideSource = true
			break
		}
	}
	if !foundOverrideSource {
		t.Fatalf("expected cli-override source in %+v", prov.Sources)
	}

	foundDryInput := false
	for _, dryInput := range overrideImport.DryInputs {
		if dryInput.Role == "cli-override" && strings.Contains(dryInput.Path, "--set image.tag=v9.9.9") {
			foundDryInput = true
			break
		}
	}
	if !foundDryInput {
		t.Fatalf("expected cli-override dry input in %+v", overrideImport.DryInputs)
	}
}
