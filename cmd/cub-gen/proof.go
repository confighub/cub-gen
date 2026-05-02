package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/confighub/cub-gen/internal/attest"
	bridgeflow "github.com/confighub/cub-gen/internal/bridge"
	"github.com/confighub/cub-gen/internal/mutationgate"
	"github.com/confighub/cub-gen/internal/proof"
	"github.com/confighub/cub-gen/internal/publish"
)

const (
	proofLogSchema          = "cub.confighub.io/proof-log/v1"
	changeBundleSchema      = "cub.confighub.io/change-bundle/v1"
	attestationRecordSchema = "cub.confighub.io/attestation/v1"
	decisionRecordSchema    = "cub.confighub.io/governed-decision-state/v1"
	mutationGateSchema      = mutationgate.SchemaVersion
)

type proofLog struct {
	SchemaVersion        string        `json:"schema_version"`
	TraceID              string        `json:"trace_id,omitempty"`
	SourceArtifactKind   string        `json:"source_artifact_kind"`
	SourceArtifactDigest string        `json:"source_artifact_digest,omitempty"`
	EventCount           int           `json:"event_count"`
	Events               []proof.Event `json:"events"`
}

func runProof(args []string) error {
	if len(args) == 0 {
		printProofUsage(os.Stderr)
		return errors.New("proof subcommand required")
	}
	switch args[0] {
	case "help", "-h", "--help":
		printProofUsage(os.Stdout)
		return nil
	case "events":
		return runProofEvents(args[1:])
	default:
		printProofUsage(os.Stderr)
		return fmt.Errorf("unknown proof subcommand: %s", args[0])
	}
}

func runProofEvents(args []string) error {
	fs := flag.NewFlagSet("proof events", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	in := fs.String("in", "-", "Evidence artifact JSON input path, or '-' for stdin")
	bundlePath := fs.String("bundle", "", "Optional bundle JSON path when input is an attestation")
	ndjson := fs.Bool("ndjson", false, "Emit one compact proof event per line")
	pretty := fs.Bool("pretty", true, "Pretty-print JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: cub-gen proof events [--in <path>|-] [--bundle <bundle.json>] [--ndjson] [--pretty]")
	}

	inputBytes, err := readFileOrStdin(*in)
	if err != nil {
		return err
	}
	log, err := buildProofLog(inputBytes, strings.TrimSpace(*bundlePath))
	if err != nil {
		return err
	}
	if *ndjson {
		return writeProofEventsNDJSON(os.Stdout, log.Events)
	}
	return writeJSON(os.Stdout, log, *pretty)
}

func buildProofLog(inputBytes []byte, bundlePath string) (proofLog, error) {
	var header struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(inputBytes, &header); err != nil {
		return proofLog{}, fmt.Errorf("parse proof input json: %w", err)
	}
	switch header.SchemaVersion {
	case changeBundleSchema:
		var bundle publish.ChangeBundle
		if err := json.Unmarshal(inputBytes, &bundle); err != nil {
			return proofLog{}, fmt.Errorf("parse bundle json: %w", err)
		}
		if err := publish.VerifyBundle(bundle); err != nil {
			return proofLog{}, err
		}
		return proofLog{
			SchemaVersion:        proofLogSchema,
			TraceID:              bundle.TraceID,
			SourceArtifactKind:   proof.ArtifactKindChangeBundle,
			SourceArtifactDigest: bundle.BundleDigest,
			EventCount:           len(bundle.ProofEvents),
			Events:               bundle.ProofEvents,
		}, nil
	case attestationRecordSchema:
		var rec attest.Record
		if err := json.Unmarshal(inputBytes, &rec); err != nil {
			return proofLog{}, fmt.Errorf("parse attestation json: %w", err)
		}
		if bundlePath == "" {
			if err := attest.VerifyRecord(rec); err != nil {
				return proofLog{}, err
			}
		} else {
			bundleBytes, err := os.ReadFile(bundlePath)
			if err != nil {
				return proofLog{}, fmt.Errorf("read bundle file: %w", err)
			}
			var bundle publish.ChangeBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return proofLog{}, fmt.Errorf("parse bundle json: %w", err)
			}
			if err := attest.VerifyRecordAgainstBundle(rec, bundle); err != nil {
				return proofLog{}, err
			}
		}
		return proofLog{
			SchemaVersion:        proofLogSchema,
			TraceID:              rec.TraceID,
			SourceArtifactKind:   proof.ArtifactKindAttestation,
			SourceArtifactDigest: rec.AttestationDigest,
			EventCount:           len(rec.ProofEvents),
			Events:               rec.ProofEvents,
		}, nil
	case decisionRecordSchema:
		var rec bridgeflow.DecisionRecord
		if err := json.Unmarshal(inputBytes, &rec); err != nil {
			return proofLog{}, fmt.Errorf("parse decision json: %w", err)
		}
		if err := bridgeflow.ValidateDecisionRecord(rec); err != nil {
			return proofLog{}, err
		}
		if len(rec.ProofEvents) == 0 {
			return proofLog{}, fmt.Errorf("decision record missing proof_events")
		}
		return proofLog{
			SchemaVersion:      proofLogSchema,
			TraceID:            rec.TraceID,
			SourceArtifactKind: proof.ArtifactKindDecision,
			EventCount:         len(rec.ProofEvents),
			Events:             rec.ProofEvents,
		}, nil
	case mutationGateSchema:
		var decision mutationgate.Decision
		if err := json.Unmarshal(inputBytes, &decision); err != nil {
			return proofLog{}, fmt.Errorf("parse mutation apply gate decision json: %w", err)
		}
		if err := mutationgate.ValidateDecisionRecord(decision); err != nil {
			return proofLog{}, err
		}
		return proofLog{
			SchemaVersion:        proofLogSchema,
			TraceID:              decision.TraceID,
			SourceArtifactKind:   proof.ArtifactKindMutationGate,
			SourceArtifactDigest: decision.DecisionDigest,
			EventCount:           len(decision.ProofEvents),
			Events:               decision.ProofEvents,
		}, nil
	default:
		return proofLog{}, fmt.Errorf("unsupported proof input schema_version %q", header.SchemaVersion)
	}
}

func readFileOrStdin(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return b, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file: %w", err)
	}
	return b, nil
}

func writeProofEventsNDJSON(out io.Writer, events []proof.Event) error {
	enc := json.NewEncoder(out)
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			return err
		}
	}
	return nil
}

func printProofUsage(out io.Writer) {
	printCommandHelp(
		out,
		"cub-gen proof: extract loggable proof records",
		[]string{
			"Proof events are JSON records that can be copied into Pilot, CI, validation, or audit logs.",
		},
		helpSection{
			Title: "COMMANDS",
			Lines: []string{
				"  events    Verify an evidence artifact and emit its proof_events",
			},
		},
		helpSection{
			Title: "EXAMPLES",
			Lines: []string{
				"  cub-gen proof events --in bundle.json",
				"  cub-gen proof events --in attestation.json --bundle bundle.json --ndjson",
				"  cub-gen proof events --in decision.json --ndjson",
				"  cub-gen proof events --in mutation-gate-decision.json --ndjson",
			},
		},
	)
}
