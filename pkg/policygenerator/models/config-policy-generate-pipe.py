import subprocess
import sys
import os
import argparse
from transformers import pipeline
import torch

# def install(package):
#     try:
#         subprocess.check_call([sys.executable, "-m", "pip", "install", package])
#     except subprocess.CalledProcessError as e:
#         print(f"Failed to install {package} with error: {e}")
#         sys.exit(1)
# install("transformers")

class PipelineSingleton:
    _instance = None

    @classmethod
    def get_instance(cls, model_id):
        if cls._instance is None:
            device = 0 if torch.cuda.is_available() else -1  # Use GPU if available, otherwise use CPU
            cls._instance = pipeline("text-generation", model=model_id, device=device)
        return cls._instance

# Function to parse command line arguments
def get_args():
    parser = argparse.ArgumentParser(description="Generate text using a transformer model.")
    parser.add_argument("--model", type=str, required=True, help="Model ID for the Hugging Face transformer.")
    parser.add_argument("--prompt", type=str, required=True, help="Prompt text to seed the generation.")
    parser.add_argument("--token", type=str, required=True, help="Hugging Face API token.")
    return parser.parse_args()

# Initialize the pipeline once and use it in the generate_text function
def generate_text(pipe, prompt):
    """Function to generate text using the specified model and input text."""
    try:
        output_text = pipe(prompt, max_new_tokens=512, do_sample=True, top_k=50, top_p=0.95, num_return_sequences=1)
        generated_text = output_text[0]['generated_text']
        # print(f"Generated text: {generated_text}", file=sys.stderr)
        return generated_text.strip()
    except Exception as e:
        sys.stderr.write(f"Error generating text: {str(e)}\n")  # Error messages to stderr
    finally:
        if pipe.device == 0:
            torch.cuda.empty_cache()

# Main function to execute the script
if __name__ == "__main__":
    args = get_args()
    os.environ['HF_TOKEN'] = args.token
    pipe = PipelineSingleton.get_instance(args.model)
    output = generate_text(pipe, args.prompt)
    sys.stdout.write(f"Generated text:\n{output}\n")