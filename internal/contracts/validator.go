package contracts

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/confighub/cub-gen/internal/model"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	generatorContractSchemaName = "schemas/generator-contract.v1.schema.json"
	provenanceSchemaName        = "schemas/provenance.v1.schema.json"
	inversePlanSchemaName       = "schemas/inverse-transform-plan.v1.schema.json"
)

//go:embed schemas/generator-contract.v1.schema.json
var generatorContractSchemaJSON string

//go:embed schemas/provenance.v1.schema.json
var provenanceSchemaJSON string

//go:embed schemas/inverse-transform-plan.v1.schema.json
var inversePlanSchemaJSON string

type compiledSchemas struct {
	generatorContract *jsonschema.Schema
	provenanceRecord  *jsonschema.Schema
	inversePlan       *jsonschema.Schema
}

var (
	schemasOnce sync.Once
	schemas     compiledSchemas
	schemasErr  error
)

// ValidateTriple validates one generator contract triple.
func ValidateTriple(contract model.GeneratorContract, provenance model.ProvenanceRecord, inversePlan model.InverseTransformPlan) error {
	if err := ValidateGeneratorContract(contract); err != nil {
		return err
	}
	if err := ValidateProvenanceRecord(provenance); err != nil {
		return err
	}
	if err := ValidateInverseTransformPlan(inversePlan); err != nil {
		return err
	}
	return nil
}

// ValidateTripleSet validates a full import triple set.
func ValidateTripleSet(contracts []model.GeneratorContract, provenance []model.ProvenanceRecord, inversePlans []model.InverseTransformPlan) error {
	if len(contracts) != len(provenance) || len(contracts) != len(inversePlans) {
		return fmt.Errorf("contract triple cardinality mismatch: contracts=%d provenance=%d inverse_plans=%d", len(contracts), len(provenance), len(inversePlans))
	}
	for i := range contracts {
		if err := ValidateTriple(contracts[i], provenance[i], inversePlans[i]); err != nil {
			return fmt.Errorf("contract triple index %d: %w", i, err)
		}
	}
	return nil
}

// ValidateGeneratorContract validates one generator contract record.
func ValidateGeneratorContract(contract model.GeneratorContract) error {
	s, err := loadSchemas()
	if err != nil {
		return err
	}
	return validateRecord("generator_contract", s.generatorContract, contract)
}

// ValidateProvenanceRecord validates one provenance record.
func ValidateProvenanceRecord(provenance model.ProvenanceRecord) error {
	s, err := loadSchemas()
	if err != nil {
		return err
	}
	return validateRecord("provenance_record", s.provenanceRecord, provenance)
}

// ValidateInverseTransformPlan validates one inverse transform plan.
func ValidateInverseTransformPlan(inversePlan model.InverseTransformPlan) error {
	s, err := loadSchemas()
	if err != nil {
		return err
	}
	return validateRecord("inverse_transform_plan", s.inversePlan, inversePlan)
}

func loadSchemas() (compiledSchemas, error) {
	schemasOnce.Do(func() {
		schemas, schemasErr = compileSchemas()
	})
	if schemasErr != nil {
		return compiledSchemas{}, fmt.Errorf("load contract schemas: %w", schemasErr)
	}
	return schemas, nil
}

func compileSchemas() (compiledSchemas, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(generatorContractSchemaName, strings.NewReader(generatorContractSchemaJSON)); err != nil {
		return compiledSchemas{}, fmt.Errorf("add %s: %w", generatorContractSchemaName, err)
	}
	if err := compiler.AddResource(provenanceSchemaName, strings.NewReader(provenanceSchemaJSON)); err != nil {
		return compiledSchemas{}, fmt.Errorf("add %s: %w", provenanceSchemaName, err)
	}
	if err := compiler.AddResource(inversePlanSchemaName, strings.NewReader(inversePlanSchemaJSON)); err != nil {
		return compiledSchemas{}, fmt.Errorf("add %s: %w", inversePlanSchemaName, err)
	}

	generatorContract, err := compiler.Compile(generatorContractSchemaName)
	if err != nil {
		return compiledSchemas{}, fmt.Errorf("compile %s: %w", generatorContractSchemaName, err)
	}
	provenanceRecord, err := compiler.Compile(provenanceSchemaName)
	if err != nil {
		return compiledSchemas{}, fmt.Errorf("compile %s: %w", provenanceSchemaName, err)
	}
	inversePlan, err := compiler.Compile(inversePlanSchemaName)
	if err != nil {
		return compiledSchemas{}, fmt.Errorf("compile %s: %w", inversePlanSchemaName, err)
	}

	return compiledSchemas{
		generatorContract: generatorContract,
		provenanceRecord:  provenanceRecord,
		inversePlan:       inversePlan,
	}, nil
}

func validateRecord(label string, schema *jsonschema.Schema, record any) error {
	payload, err := toJSONAny(record)
	if err != nil {
		return fmt.Errorf("%s marshal for schema validation: %w", label, err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("%s schema validation failed: %s", label, normalizeValidationError(err))
	}
	return nil
}

func toJSONAny(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeValidationError(err error) string {
	msg := strings.TrimSpace(err.Error())
	msg = strings.Join(strings.Fields(msg), " ")
	return msg
}
