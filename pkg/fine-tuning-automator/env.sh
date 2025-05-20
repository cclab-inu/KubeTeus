#!/bin/bash


# Environment doesn't exist, create and activate it
echo "Creating and activating 'autotrain' environment..."
echo "Updating system and installing required packages..."
sudo apt update
sudo apt install curl -y

echo "Downloading and installing Anaconda..."
curl -o anaconda.sh https://repo.anaconda.com/archive/Anaconda3-2022.10-Linux-x86_64.sh
bash anaconda.sh -b -p $HOME/anaconda3
echo "export PATH=$HOME/anaconda3/bin:$PATH" >> $HOME/.bashrc
source $HOME/.bashrc
rm -rf anaconda.sh

echo "Checking Anaconda installation..."
conda -V

echo "Creating 'autotrain' environment..."
conda create -n autotrain python=3.10 -y
exit
conda activate autotrain

echo "Installing required Python packages..."
pip install autotrain-advanced wandb
conda install pytorch torchvision torchaudio pytorch-cuda=12.1 -c pytorch -c nvidia
conda install -c "nvidia/label/cuda-12.1.0" cuda-nvcc -y


echo "Upgrading Hugging Face Hub and installing Transformers..."
pip install --upgrade huggingface_hub
pip install --force-reinstall huggingface_hub transformers

echo "Environment setup complete."