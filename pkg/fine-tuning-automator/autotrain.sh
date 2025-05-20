#!/bin/bash

# Ensure the virtual environment is set up
export PATH=$PATH:/home/cclab/anaconda3/envs/autotrain/bin
source $HOME/.bashrc
source ~/anaconda3/etc/profile.d/conda.sh

conda info --env | grep autotrain &>/dev/null
if [ $? -ne 0 ]; then
    # Environment doesn't exist, run setup script
    ./env.sh
fi

# Environment exists, activate it
conda activate autotrain

# Load configuration
CONFIG_PATH="./conf/custom.yaml"
if [ ! -f "$CONFIG_PATH" ]; then
    echo "Configuration file not found at $CONFIG_PATH"
    exit 1
fi

# Load configuration
CONFIG_PATH="./conf/custom.yaml"
if [ ! -f "$CONFIG_PATH" ]; then
    echo "Configuration file not found at $CONFIG_PATH"
    exit 1
fi

# Extracting values using yq
TOKEN=$(yq e '.user.huggingface-token' $CONFIG_PATH)
PROJECT_NAME=$(yq e '.parameter.project-name' $CONFIG_PATH)
DATA_PATH=$(yq e '.parameter.data-path' $CONFIG_PATH)
LEARNING_RATE=$(yq e '.parameter.learning-rate' $CONFIG_PATH)
TRAIN_BATCH=$(yq e '.parameter.train-batch' $CONFIG_PATH)
TRAIN_EPOCHS=$(yq e '.parameter.train-epochs' $CONFIG_PATH)
MODEL_MAX_LENGTH=$(yq e '.parameter.model-max-length' $CONFIG_PATH)

# Run the fine-tuning command
autotrain llm --train \
  --token "$TOKEN" \
  --project-name "$PROJECT_NAME" \
  --model "$1" \
  --data-path "$DATA_PATH" \
  --text_column "text" \
  --lr "$LEARNING_RATE" \
  --batch-size "$TRAIN_BATCH" \
  --epochs "$TRAIN_EPOCHS" \
  --model-max-length "$MODEL_MAX_LENGTH" \
  --trainer sft

if [ $? -ne 0 ]; then
    echo "Fine-tuning failed"
    exit 1
fi

mv "$PROJECT_NAME" "./projects/$PROJECT_NAME"
echo "Fine-tuning completed successfully"