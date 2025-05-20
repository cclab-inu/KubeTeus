package promptprocessor

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func UserEnitityClassifier(ctx context.Context, k8sClient client.Client, intent string) ([]utils.Entity, []string, utils.NetInfo, error) {
	logger := log.FromContext(ctx)

	basePath, err := filepath.Abs("pkg/promptprocessor/classifier")
	if err != nil {
		logger.Error(err, "error determining absolute path")
		return nil, nil, utils.NetInfo{}, err
	}
	scriptPath := filepath.Join(basePath, "predict.py")
	args := []string{scriptPath, "--text", intent}
	cmd := exec.Command("python3", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error(err, "error executing python script", "output", string(output))
		return nil, nil, utils.NetInfo{}, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		logger.Error(err, "failed to unmarshal prediction result")
		return nil, nil, utils.NetInfo{}, err
	}

	predictedEntitiesRaw := result["entities"].([]interface{})
	predictedEntities := make([]utils.Entity, len(predictedEntitiesRaw))
	for i, entity := range predictedEntitiesRaw {
		entityMap := entity.(map[string]interface{})
		predictedEntities[i] = utils.Entity{
			Text: entityMap["text"].(string),
			Type: entityMap["type"].(string),
		}
	}

	finalIntent, netInfo, otherInfo, err := UserPromptProcessor(ctx, k8sClient, intent, predictedEntities)
	if err != nil {
		logger.Error(err, "failed to generate final intent")
		return nil, nil, utils.NetInfo{}, err
	}

	logger.Info("Intent analysis completed", "Intent.Entity", predictedEntities, "Intent.Prompts", finalIntent, "Pod.Info", netInfo, "OtherPod.Info", otherInfo)
	return predictedEntities, finalIntent, netInfo, nil
}
