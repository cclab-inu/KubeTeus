package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
)

var (
	ciliumNetworkPolicySchema *gojsonschema.Schema
)

func init() {
	// Load CiliumNetworkPolicy schema
	ciliumSchemaLoader := gojsonschema.NewStringLoader(loadSchema("cilium_network_policy_crd.json"))
	schema, err := gojsonschema.NewSchema(ciliumSchemaLoader)
	if err != nil {
		panic(fmt.Sprintf("failed to load CiliumNetworkPolicy schema: %v", err))
	}
	ciliumNetworkPolicySchema = schema
}

func loadSchema(schemaFile string) string {
	// Assuming the file is in the same directory
	path, err := filepath.Abs(filepath.Join("./pkg/validator/schemas", schemaFile))
	if err != nil {
		panic(fmt.Sprintf("failed to get absolute path for schema file: %v", err))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to read schema file: %v", err))
	}
	return string(data)
}

func CRDValidator(policy map[string]interface{}) []error {
	var validationErrors []error

	var schema *gojsonschema.Schema
	var policyJSON []byte
	var err error

	kind, ok := policy["kind"].(string)
	if !ok {
		validationErrors = append(validationErrors, fmt.Errorf("missing policy kind"))
		return validationErrors
	}

	switch kind {
	case "CiliumNetworkPolicy":
		schema = ciliumNetworkPolicySchema
	default:
		validationErrors = append(validationErrors, fmt.Errorf("unknown policy kind: %s", kind))
		return validationErrors
	}

	policyJSON, err = json.Marshal(policy)
	if err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("failed to marshal policy: %v", err))
		return validationErrors
	}

	documentLoader := gojsonschema.NewStringLoader(string(policyJSON))
	result, err := schema.Validate(documentLoader)
	if err != nil {
		validationErrors = append(validationErrors, fmt.Errorf("failed to validate policy: %v", err))
		return validationErrors
	}

	if !result.Valid() {
		for _, desc := range result.Errors() {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("%s", desc.String()),
			)
		}
	}

	return validationErrors
}
