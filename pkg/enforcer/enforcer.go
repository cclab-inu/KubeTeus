package enforcer

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Enforcer(ctx context.Context, k8sClient client.Client, policies interface{}) error {
	policySlice, ok := policies.([]interface{})
	if !ok {
		return fmt.Errorf("failed to assert policies as a slice of interfaces")
	}

	for _, policyInterface := range policySlice {
		unstructuredPolicy, err := convertToUnstructured(&policyInterface)
		if err != nil {
			return fmt.Errorf("failed to convert policy to unstructured: %v", err)
		}

		if err := applyPolicy(ctx, k8sClient, unstructuredPolicy); err != nil {
			return err
		}
	}

	return nil
}

func convertToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert object to unstructured: %v", err)
	}
	return &unstructured.Unstructured{Object: unstructuredMap}, nil
}

func applyPolicy(ctx context.Context, k8sClient client.Client, policy *unstructured.Unstructured) error {
	if err := k8sClient.Create(ctx, policy); err != nil {
		if errors.IsAlreadyExists(err) {
			if err := k8sClient.Update(ctx, policy); err != nil {
				return fmt.Errorf("failed to update existing policy: %v", err)
			}
		} else {
			return fmt.Errorf("failed to create policy: %v", err)
		}
	}

	return nil
}
