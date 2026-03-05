package model

type GeneratorKind string

const (
	GeneratorHelm       GeneratorKind = "helm"
	GeneratorScore      GeneratorKind = "score"
	GeneratorSpringBoot GeneratorKind = "springboot"
)

type GeneratorDetection struct {
	ID         string        `json:"id"`
	Kind       GeneratorKind `json:"kind"`
	Profile    string        `json:"profile"`
	Name       string        `json:"name"`
	Root       string        `json:"root"`
	Inputs     []string      `json:"inputs"`
	Confidence float64       `json:"confidence"`
}

type DetectionResult struct {
	Repo       string               `json:"repo"`
	Ref        string               `json:"ref"`
	DetectedAt string               `json:"detected_at"`
	Generators []GeneratorDetection `json:"generators"`
}

type UnitRef struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Layer string `json:"layer"`
}

type UnitLink struct {
	DryUnitID       string `json:"dry_unit_id"`
	WetUnitID       string `json:"wet_unit_id"`
	GeneratorUnitID string `json:"generator_unit_id"`
}

type GeneratorInput struct {
	Name      string `json:"name"`
	SchemaRef string `json:"schema_ref"`
	Required  bool   `json:"required"`
}

type GeneratorContract struct {
	SchemaVersion string           `json:"schema_version"`
	GeneratorID   string           `json:"generator_id"`
	Name          string           `json:"name"`
	Kind          string           `json:"kind"`
	Profile       string           `json:"profile"`
	Version       string           `json:"version"`
	SourceRepo    string           `json:"source_repo"`
	SourceRef     string           `json:"source_ref"`
	SourcePath    string           `json:"source_path"`
	Inputs        []GeneratorInput `json:"inputs"`
	OutputFormat  string           `json:"output_format"`
	Transport     string           `json:"transport"`
	Capabilities  []string         `json:"capabilities"`
	Deterministic bool             `json:"deterministic"`
}

type SourceRef struct {
	Role     string `json:"role"`
	URI      string `json:"uri"`
	Revision string `json:"revision"`
	Path     string `json:"path"`
}

type OutputRef struct {
	Role   string `json:"role"`
	URI    string `json:"uri"`
	Digest string `json:"digest"`
}

type RenderedObjectLineage struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Namespace     string `json:"namespace,omitempty"`
	SourcePath    string `json:"source_path,omitempty"`
	SourceDryPath string `json:"source_dry_path,omitempty"`
}

type FieldOrigin struct {
	DryPath    string  `json:"dry_path"`
	WetPath    string  `json:"wet_path"`
	SourcePath string  `json:"source_path"`
	Transform  string  `json:"transform"`
	Confidence float64 `json:"confidence"`
}

type InverseEditPointer struct {
	WetPath    string  `json:"wet_path"`
	DryPath    string  `json:"dry_path"`
	Owner      string  `json:"owner"`
	EditHint   string  `json:"edit_hint"`
	Confidence float64 `json:"confidence"`
}

type ProvenanceRecord struct {
	SchemaVersion       string                  `json:"schema_version"`
	ProvenanceID        string                  `json:"provenance_id"`
	ChangeID            string                  `json:"change_id"`
	GeneratorID         string                  `json:"generator_id"`
	GeneratorName       string                  `json:"generator_name"`
	GeneratorProfile    string                  `json:"generator_profile"`
	Version             string                  `json:"version"`
	InputDigest         string                  `json:"input_digest"`
	Sources             []SourceRef             `json:"sources"`
	Outputs             []OutputRef             `json:"outputs"`
	ChartPath           string                  `json:"chart_path,omitempty"`
	ValuesPaths         []string                `json:"values_paths,omitempty"`
	RenderedLineage     []RenderedObjectLineage `json:"rendered_object_lineage,omitempty"`
	FieldOriginMap      []FieldOrigin           `json:"field_origin_map"`
	InverseEditPointers []InverseEditPointer    `json:"inverse_edit_pointers"`
	RenderedAt          string                  `json:"rendered_at"`
}

type InversePatch struct {
	Operation      string  `json:"op"`
	DryPath        string  `json:"dry_path"`
	WetPath        string  `json:"wet_path"`
	EditableBy     string  `json:"editable_by"`
	Confidence     float64 `json:"confidence"`
	RequiresReview bool    `json:"requires_review"`
	Reason         string  `json:"reason"`
}

type InverseTransformPlan struct {
	SchemaVersion string         `json:"schema_version"`
	PlanID        string         `json:"plan_id"`
	ChangeID      string         `json:"change_id"`
	SourceKind    string         `json:"source_kind"`
	SourceRef     string         `json:"source_ref"`
	TargetUnitID  string         `json:"target_unit_id"`
	Status        string         `json:"status"`
	Patches       []InversePatch `json:"patches"`
	CreatedAt     string         `json:"created_at"`
}

type DryInputRef struct {
	GeneratorID string `json:"generator_id"`
	Profile     string `json:"profile"`
	Role        string `json:"role"`
	Owner       string `json:"owner"`
	Path        string `json:"path"`
	Required    bool   `json:"required"`
}

type WetManifestTarget struct {
	GeneratorID   string `json:"generator_id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Owner         string `json:"owner"`
	Namespace     string `json:"namespace,omitempty"`
	SourceDryPath string `json:"source_dry_path,omitempty"`
}

type ImportResult struct {
	Repo               string                 `json:"repo"`
	Ref                string                 `json:"ref"`
	Space              string                 `json:"space"`
	ChangeID           string                 `json:"change_id"`
	ImportedAt         string                 `json:"imported_at"`
	Detection          DetectionResult        `json:"detection"`
	Units              []UnitRef              `json:"units"`
	Links              []UnitLink             `json:"links"`
	GeneratorContracts []GeneratorContract    `json:"generator_contracts"`
	Provenance         []ProvenanceRecord     `json:"provenance"`
	InversePlans       []InverseTransformPlan `json:"inverse_transform_plans"`
	DryInputs          []DryInputRef          `json:"dry_inputs"`
	WetManifestTargets []WetManifestTarget    `json:"wet_manifest_targets"`
}
