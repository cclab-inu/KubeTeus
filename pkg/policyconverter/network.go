package policyconverter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

func processNetworkPolicies(ctx context.Context, podInfo utils.NetInfo, entities []utils.Entity, policyStrings []string) ([]interface{}, []error) {
	var processedPolicies []interface{}
	var validationErrors []error
	logger := log.FromContext(ctx)

	for _, policyStr := range policyStrings {
		cleanPolicyStr := cleanPolicyString(policyStr)
		logger.Info("Policy Processing started", "Clean", cleanPolicyStr)
		var genericPolicy map[string]interface{}
		jsonBytes, err := yaml.YAMLToJSON([]byte(cleanPolicyStr))
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to convert YAML to JSON: %v", err))
			continue
		}
		if err := json.Unmarshal(jsonBytes, &genericPolicy); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("failed to unmarshal policy: %v", err))
			continue
		}

		spec, err := extractMinimalNetworkSpecFromPolicy(genericPolicy)
		if err != nil {
			validationErrors = append(validationErrors, err)
			continue
		}

		crd, err := createMinimalNetworkCRDFromPolicy(spec, podInfo, entities)
		if err != nil {
			validationErrors = append(validationErrors, err)
			continue
		}
		processedPolicies = append(processedPolicies, crd)
	}
	return processedPolicies, validationErrors
}

func extractMinimalNetworkSpecFromPolicy(policy map[string]interface{}) (map[string]interface{}, error) {
	spec := make(map[string]interface{})

	specData, ok := policy["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("spec not found in policy")
	}

	if endpointSelector, ok := specData["endpointSelector"].(map[string]interface{}); ok {
		spec["endpointSelector"] = endpointSelector
	}
	if ingress, ok := specData["ingress"].([]interface{}); ok {
		ingressRules, err := convertMinimalIngressRules(ingress)
		if err != nil {
			return nil, fmt.Errorf("failed to convert ingress rules: %v", err)
		}
		spec["ingress"] = ingressRules
	}
	if egress, ok := specData["egress"].([]interface{}); ok {
		egressRules, err := convertMinimalEgressRules(egress)
		if err != nil {
			return nil, fmt.Errorf("failed to convert egress rules: %v", err)
		}
		spec["egress"] = egressRules
	}

	return spec, nil
}

func convertMinimalIngressRules(ingress []interface{}) ([]map[string]interface{}, error) {
	var ingressRules []map[string]interface{}
	for _, rule := range ingress {
		if ruleMap, ok := rule.(map[string]interface{}); ok {
			minimalRule := make(map[string]interface{})
			if ports, ok := ruleMap["ports"].([]interface{}); ok {
				minimalRule["ports"] = ports
			}
			if protocol, ok := ruleMap["protocol"].(string); ok {
				minimalRule["protocol"] = protocol
			}
			ingressRules = append(ingressRules, minimalRule)
		} else {
			return nil, fmt.Errorf("invalid ingress rule format")
		}
	}
	return ingressRules, nil
}

func convertMinimalEgressRules(egress []interface{}) ([]map[string]interface{}, error) {
	var egressRules []map[string]interface{}
	for _, rule := range egress {
		if ruleMap, ok := rule.(map[string]interface{}); ok {
			minimalRule := make(map[string]interface{})
			if ports, ok := ruleMap["ports"].([]interface{}); ok {
				minimalRule["ports"] = ports
			}
			if protocol, ok := ruleMap["protocol"].(string); ok {
				minimalRule["protocol"] = protocol
			}
			egressRules = append(egressRules, minimalRule)
		} else {
			return nil, fmt.Errorf("invalid egress rule format")
		}
	}
	return egressRules, nil
}

func createMinimalNetworkCRDFromPolicy(spec map[string]interface{}, podInfo utils.NetInfo, entities []utils.Entity) (interface{}, error) {
	var cnp ciliumv2.CiliumNetworkPolicy
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           &cnp,
		TagName:          "json",
		WeaklyTypedInput: true,
	}

	_, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, fmt.Errorf("error initializing decoder: %v", err)
	}

	cnp.ObjectMeta = metav1.ObjectMeta{
		Name:      fmt.Sprintf("cnp-%s-%s-%s", getActionFromEntities(entities), podInfo.Name, getPortFromEntities(entities)),
		Namespace: podInfo.Namespace,
	}

	cnp.APIVersion = "cilium.io/v2"
	cnp.Kind = "CiliumNetworkPolicy"
	cnp.Spec = &api.Rule{}
	err = mapstructure.Decode(spec, cnp.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to decode spec: %v", err)
	}

	return cnp, nil
}

func getActionFromEntities(entities []utils.Entity) string {
	for _, entity := range entities {
		if entity.Type == "ACTION" {
			return strings.ToLower(entity.Text)
		}
	}
	return ""
}

func getPortFromEntities(entities []utils.Entity) string {
	for _, entity := range entities {
		if entity.Type == "PORT" {
			return entity.Text
		}
	}
	return ""
}
