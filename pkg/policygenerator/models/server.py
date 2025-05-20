import subprocess
import sys
import os
import argparse
from transformers import pipeline
import torch
from flask import Flask, request, jsonify, Response

torch.cuda.empty_cache()

app = Flask(__name__)

class PipelineSingleton:
    _instance = None
    @classmethod
    def get_instance(cls, model_id):
        if cls._instance is None:
            model = pipeline("text-generation", model=model_id, device=-1, batch_size=1)
            cls._instance = model

        return cls._instance

def get_args():
    parser = argparse.ArgumentParser(description="Run a Flask server for text generation using a transformer model.")
    parser.add_argument("--model", type=str, required=True, help="Model ID for the Hugging Face transformer.")
    parser.add_argument("--port", type=int, required=True, help="Port for the Flask server.")
    parser.add_argument("--token", type=str, required=True, help="Hugging Face API token.")
    return parser.parse_args()

def generate_text(pipe, prompt):
    """Function to generate text using the specified model and input text."""
    try:
        torch.cuda.empty_cache()
        if torch.cuda.is_available():
            device = torch.device('cuda')
            pipe.model.to(device)
        else:
            device = torch.device('cpu')
        input_ids = pipe.tokenizer(prompt, return_tensors='pt').input_ids.to(device)
        attention_mask = pipe.tokenizer(prompt, return_tensors='pt').attention_mask.to(device)

        output = pipe.model.generate(input_ids, attention_mask=attention_mask, max_new_tokens=512, do_sample=True, top_k=50, top_p=0.95, num_return_sequences=1)
        generated_text = pipe.tokenizer.decode(output[0], skip_special_tokens=True)

        if torch.cuda.is_available():
            pipe.model.to('cpu')

        return generated_text.strip()  # Strip leading and trailing whitespace
    except Exception as e:
        return str(e)
    finally:
        torch.cuda.empty_cache()

@app.route('/generate', methods=['POST'])
def generate():
    data = request.get_json()
    if not data or 'prompt' not in data:
        return jsonify({"error": "Invalid request, missing 'prompt' field"}), 400

    prompt = data['prompt']
    generated_text = generate_text(pipe, prompt)
    
    # Log the generated text for debugging
    print(f"Generated text: {generated_text}", file=sys.stderr)
    
    # Directly return the raw generated text
    return Response(generated_text, status=200, mimetype='text/plain')

if __name__ == "__main__":
    args = get_args()
    os.environ['HF_TOKEN'] = args.token
    pipe = PipelineSingleton.get_instance(args.model)
    
    app.run(host='0.0.0.0', port=args.port)
