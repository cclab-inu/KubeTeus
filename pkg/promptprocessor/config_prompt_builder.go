package promptprocessor

import (
	"context"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigPromptBuilder(ctx context.Context, k8sClient client.Client, netInfo *utils.NetInfo) (utils.Prompt, error) {
	db, err := connectToDB()
	if err != nil {
		return utils.Prompt{}, err
	}
	defer db.Close()

	netPrompts, err := GenerateCiliumPrompts(ctx, k8sClient, netInfo)
	if err != nil {
		return utils.Prompt{}, err
	}

	return utils.Prompt{Network: netPrompts}, nil
}
