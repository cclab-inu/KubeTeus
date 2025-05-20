// validator.go
package validator

import (
	"context"
	"encoding/json"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Validator(ctx context.Context, k8sClient client.Client, logger logr.Logger, podInfo utils.NetInfo, entities []utils.Entity, policies interface{}) ([]error, error) {
	var validationErrors []error
	logger.Info("[CRD Validator] CRD Validator started")
	for _, policy := range policies.([]interface{}) {
		// Convert policy to map[string]interface{}
		policyMap, err := toMap(policy)
		if err != nil {
			validationErrors = append(validationErrors, err)
			continue
		}
		if errs := CRDValidator(policyMap); len(errs) > 0 {
			validationErrors = append(validationErrors, errs...)
		}
	}
	if len(validationErrors) > 0 {
		return validationErrors, nil
	} else {
		logger.Info("[CRD Validator] Policy CRD syntax validated", "Result", "No problems")
	}

	logger.Info("[Resource Validator] Resource Validator started")
	resourceValidationErrors := ResourceValidator(ctx, k8sClient, logger, policies)
	if len(resourceValidationErrors) > 0 {
		validationErrors = append(validationErrors, resourceValidationErrors...)
	}
	logger.Info("[Resource Validator] Resource Validator done")

	logger.Info("[Property Validator] Property Validator started")
	propertyValidationErrors := PropertyValidator(ctx, k8sClient, policies)
	if len(propertyValidationErrors) > 0 {
		validationErrors = append(validationErrors, propertyValidationErrors...)
	}
	logger.Info("[Property Validator] Property Validator done")

	return validationErrors, nil
}

func toMap(policy interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	var policyMap map[string]interface{}
	err = json.Unmarshal(data, &policyMap)
	if err != nil {
		return nil, err
	}

	return policyMap, nil
}
