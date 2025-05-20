import subprocess
import sys
import os
import argparse
from transformers import AutoModelForCausalLM, AutoTokenizer
import torch

def install(package):
    try:
        subprocess.check_call([sys.executable, "-m", "pip", "install", package])
    except subprocess.CalledProcessError as e:
        print(f"Failed to install {package} with error: {e}")
        sys.exit(1)
#install("transformers")

class ModelSingleton:
    _instance = None

    @classmethod
    def get_instance(cls, model_id):
        if cls._instance is None:
            # Check if GPU is available, otherwise use CPU
            device = "cuda" if torch.cuda.is_available() else "cpu"
            model = AutoModelForCausalLM.from_pretrained(model_id, torch_dtype=torch.bfloat16 if torch.cuda.is_available() else torch.float32).to(device)
            tokenizer = AutoTokenizer.from_pretrained(model_id)
            cls._instance = (model, tokenizer, device)
        return cls._instance

def get_args():
    parser = argparse.ArgumentParser(description="Generate text using a transformer model.")
    parser.add_argument("--model", type=str, required=True, help="Model ID for the Hugging Face transformer.")
    parser.add_argument("--prompt", type=str, required=True, help="Prompt text to seed the generation.")
    parser.add_argument("--token", type=str, required=True, help="Hugging Face API token.")
    return parser.parse_args()

def generate_text(model, tokenizer, prompt):
    """Function to generate text using the specified model and input text."""
    try:
        # Tokenize input
        inputs = tokenizer(prompt, return_tensors="pt", padding=True, truncation=True)
        input_ids = inputs.input_ids.to(model.device)
        attention_mask = inputs.attention_mask.to(model.device)

        # Generate text
        output = model.generate(input_ids, attention_mask=attention_mask, max_new_tokens=256, do_sample=True, top_k=50, top_p=0.95, num_return_sequences=1)

        # Decode generated output
        decoded_output = tokenizer.decode(output[0], skip_special_tokens=True)
        print(decoded_output)
    finally:
        # Ensure GPU memory is cleared
        if model.device == "cuda":
            torch.cuda.empty_cache()

if __name__ == "__main__":
    args = get_args()
    model, tokenizer, device = ModelSingleton.get_instance(args.model)
    generate_text(model, tokenizer, args.prompt)