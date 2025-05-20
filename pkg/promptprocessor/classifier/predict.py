import torch
from transformers import BertTokenizer, BertModel
from model import BertForEntityClassification
from custom_tokenizer import CustomBertTokenizer
import argparse
import re
import json

index_to_entity_tag = {0: 'POLICY', 1: 'LABEL', 2: 'POD_NAME', 3: 'NAMESPACE', 4: 'ACTION',
                       5: 'TRAFFIC_DIRECTION', 6: 'CIDR', 7: 'PORT', 8: 'PROTOCOL',
                       9: 'PATH', 10: 'FILE', 11: 'HTTP_PATH', 12: 'HTTP_METHOD', 
                       13: 'FQDN', 14: 'ENDPOINT', 15: 'TRASH'}

POLICY_PATTERN = re.compile(r'\b(CiliumNetworkPolicy|network policy|security policy|KubeArmorHostPolicy|KubeArmorPolicy|policy|Policy)\b', re.IGNORECASE)
POD_PATTERN = re.compile(r"'\s*([^']+?)\s*'\s+(pod|pods)\b|\b([^'\s]+)\s+(pod|pods)|(pod|pods)\b", re.IGNORECASE)
PORT_PATTERN = re.compile(r'\bport (\d+)\b|\b(\d+) port\b|\bport\b|\'(\d+)\'', re.IGNORECASE)
ACTION_PATTERN = re.compile(r'\b(allow|allows|deny|denies|block|blocks|audit|restrict|prevent|prevents|blocking)\b', re.IGNORECASE)
ENDPOINT_PATTERN = re.compile(r'\b(endpoints|endpoint)\b', re.IGNORECASE)
PROTOCOL_PATTERN = re.compile(r'\b(TCP|UDP|ICMP)\b', re.IGNORECASE)
TRAFFIC_DIRECTION_PATTERN = re.compile(r'\b(ingress|egress|income|incoming|inbound|outcome|outcoming|outbound)\b', re.IGNORECASE)
PATH_PATTERN = re.compile(r'(/[a-zA-Z_][-a-zA-Z0-9_/.]*[^/])')
FILE_PATTERN = re.compile(r'\b([a-zA-Z_][-a-zA-Z0-9_/.]*\.[a-zA-Z0-9]+)\b', re.IGNORECASE)

class Predictor:
    def __init__(self, entity_model_path, device):
        self.device = device
        
        bert_model = BertModel.from_pretrained('bert-base-uncased')
        self.entity_tokenizer = CustomBertTokenizer.from_pretrained('bert-base-uncased')

        self.entity_model = BertForEntityClassification(n_entity_types=len(index_to_entity_tag), bert_model=bert_model)
        self.entity_model.load_state_dict(torch.load(entity_model_path, map_location=device))
        self.entity_model.to(self.device)

    def predict_entities(self, text, max_length=512):
        self.entity_model.eval()

        inputs = self.entity_tokenizer.encode_plus_custom(
            text,
            add_special_tokens=True,
            max_length=max_length,
            padding='max_length',
            truncation=True,
            return_tensors="pt"
        )

        input_ids = inputs['input_ids'].to(self.device)
        attention_mask = inputs['attention_mask'].to(self.device)

        with torch.no_grad():
            logits = self.entity_model(input_ids, attention_mask)
            predicted_entities = logits.argmax(dim=2).squeeze().tolist()

        tokens = self.entity_tokenizer.convert_ids_to_tokens(input_ids.squeeze().tolist())

        filtered_entities = []
        current_word = ""
        current_tag = None

        for token, tag in zip(tokens, predicted_entities):
            if token in ["[PAD]", "[CLS]", "[SEP]"]:
                continue
            if token.startswith("##"):
                current_word += token[2:]
            else:
                if current_word:
                    filtered_entities.append((current_word, index_to_entity_tag[current_tag]))
                current_word = token
                current_tag = tag

        if current_word:
            filtered_entities.append((current_word, index_to_entity_tag[current_tag]))

        # Handle LABEL and PORT tokens post-processing
        final_entities = []
        i = 0
        while i < len(filtered_entities):
            if filtered_entities[i][1] == 'LABEL':
                combined_word = filtered_entities[i][0]
                j = i + 1
                while j < len(filtered_entities) and filtered_entities[j][1] == 'LABEL':
                    combined_word += " " + filtered_entities[j][0]
                    j += 1

                # Check if the next token is POLICY and there are exactly two LABELs before it
                if j < len(filtered_entities) and filtered_entities[j][1] == 'POLICY' and (j - i) == 2:
                    combined_word += " " + filtered_entities[j][0]
                    combined_word = combined_word.replace(" :", ":")
                    final_entities.append((combined_word, 'LABEL'))
                    i = j + 1
                else:
                    if ':' not in combined_word:
                        combined_word += " pod"
                    final_entities.append((combined_word, 'LABEL'))
                    i += 1
            elif filtered_entities[i][1] == 'PORT':
                port_word = filtered_entities[i][0]
                port_number = None
                j = i + 1
                # Look ahead for numbers
                if j < len(filtered_entities) and filtered_entities[j][1] != 'PORT':
                    try:
                        port_number = int(filtered_entities[j][0])
                        final_entities.append((f"{port_word} {port_number}", 'PORT'))
                        i = j + 1
                        continue
                    except ValueError:
                        pass
                # Combine consecutive PORT tokens
                while j < len(filtered_entities) and filtered_entities[j][1] == 'PORT':
                    port_word += " " + filtered_entities[j][0]
                    j += 1
                final_entities.append((port_word, 'PORT'))
                i = j
            elif filtered_entities[i][1] == 'POLICY':
                if (i + 1 < len(filtered_entities) and 
                    filtered_entities[i + 1][1] == 'POD_NAME' and 
                    not POLICY_PATTERN.match(filtered_entities[i][0])):
                    final_entities.append((filtered_entities[i][0], 'POD_NAME'))
                else:
                    final_entities.append(filtered_entities[i])
                i += 1
            else:
                final_entities.append(filtered_entities[i])
                i += 1

        # Post-process ACTION, ENDPOINT, PROTOCOL, and TRAFFIC_DIRECTION tokens
        for i in range(len(final_entities)):
            word, tag = final_entities[i]
            if ACTION_PATTERN.match(word) and tag != 'ACTION':
                final_entities[i] = (word, 'ACTION')
            elif ENDPOINT_PATTERN.match(word) and tag != 'ENDPOINT':
                final_entities[i] = (word, 'ENDPOINT')
            elif PROTOCOL_PATTERN.match(word) and tag != 'PROTOCOL':
                final_entities[i] = (word, 'PROTOCOL')
            elif TRAFFIC_DIRECTION_PATTERN.match(word) and tag != 'TRAFFIC_DIRECTION':
                final_entities[i] = (word, 'TRAFFIC_DIRECTION')

        for i in range(len(final_entities)):
            word, tag = final_entities[i]
            if tag in {'POLICY', 'FILE', 'FQDN'} and not POLICY_PATTERN.match(word):
                # Check if it matches PATH_PATTERN or FILE_PATTERN in the original text
                path_match = PATH_PATTERN.search(text)
                file_match = FILE_PATTERN.search(text)
                if path_match and word in path_match.group(0):
                    final_entities[i] = (word, 'PATH')
                elif file_match and word in file_match.group(0):
                    final_entities[i] = (word, 'FILE')

        # Post-process to handle consecutive POD_NAME tokens
        i = 0
        while i < len(final_entities) - 1:
            if final_entities[i][1] == 'POD_NAME' and final_entities[i + 1][1] == 'POD_NAME':
                combined_pod_name = f"{final_entities[i][0]} {final_entities[i + 1][0]}"
                final_entities[i] = (combined_pod_name, 'POD_NAME')
                del final_entities[i + 1]
            else:
                i += 1
                
        # Post-process to handle '/' between PORT and PROTOCOL
        i = 0
        while i < len(final_entities) - 2:
            if (final_entities[i][1] == 'PORT' and 
                final_entities[i+1][0] == '/' and 
                final_entities[i+2][1] == 'PROTOCOL'):
                final_entities[i+1] = (final_entities[i+1][0], 'TRASH')
            i += 1

        # Remove TRASH entities
        final_entities = [entity for entity in final_entities if entity[1] != 'TRASH']

        # Additional processing for finding POD names in the format '[POD_NAME] pod'
        for i in range(len(final_entities) - 1):
            if (final_entities[i][1] != 'POD_NAME' and 
                POD_PATTERN.match(final_entities[i][0] + " " + final_entities[i + 1][0])):
                final_entities[i] = (final_entities[i][0], 'POD_NAME')
                final_entities[i + 1] = (final_entities[i + 1][0], 'POD_NAME')

        # Ensure POLICY_PATTERN matches only get tagged as POLICY
        for i in range(len(final_entities)):
            if POLICY_PATTERN.match(final_entities[i][0]):
                final_entities[i] = (final_entities[i][0], 'POLICY')
            elif final_entities[i][1] == 'POLICY' and not POLICY_PATTERN.match(final_entities[i][0]):
                final_entities[i] = (final_entities[i][0], 'LABEL')

        # Add ACTION entities if not present
        if not any(tag == 'ACTION' for _, tag in final_entities):
            words = text.split()
            for word in words:
                if ACTION_PATTERN.match(word):
                    final_entities.append((word, 'ACTION'))

        # Combine PATH tokens into a single PATH entity
        path_entities = [e for e in final_entities if e[1] == 'PATH']
        if path_entities:
            combined_path = "".join([e[0] for e in path_entities]).replace(" ", "")
            match = PATH_PATTERN.search(text)
            if match and match.group(0).replace(" ", "") != combined_path:
                combined_path = match.group(0).replace(" ", "")
            final_entities = [e for e in final_entities if e[1] != 'PATH']
            final_entities.append((combined_path, 'PATH'))

        # Combine consecutive POD_NAME tokens into a single POD_NAME entity with two words
        i = 0
        while i < len(final_entities) - 1:
            if final_entities[i][1] == 'POD_NAME' and final_entities[i + 1][1] == 'POD_NAME':
                combined_pod_name = f"{final_entities[i][0]} {final_entities[i + 1][0]}"
                final_entities[i] = (combined_pod_name, 'POD_NAME')
                del final_entities[i + 1]
            else:
                i += 1

        # Combine consecutive LABEL tokens into a single LABEL entity
        i = 0
        while i < len(final_entities) - 2:
            if (final_entities[i][1] == 'LABEL' and final_entities[i + 1][1] == 'LABEL' and final_entities[i + 2][1] == 'LABEL'):
                combined_label = f"{final_entities[i][0]}{final_entities[i + 1][0]} {final_entities[i + 2][0]}"
                final_entities[i] = (combined_label, 'LABEL')
                del final_entities[i + 1]
                del final_entities[i + 1]
            else:
                i += 1

        # Additional step to split combined POD_NAME entities
        final_entities = self.split_combined_pod_names(final_entities)

        return final_entities

    def split_combined_pod_names(self, entities):
        """Splits combined POD_NAME entities like 'nginx2 pod nginx1 pod' into separate entities."""
        pod_name_pattern = re.compile(r'(\w+-\w+-\w+-\w+|\w+)\s+pod', re.IGNORECASE)
        split_entities = []
        for word, tag in entities:
            if tag == 'POD_NAME':
                matches = pod_name_pattern.findall(word)
                if matches:
                    split_entities.extend([(match.strip(), 'POD_NAME') for match in matches])
                else:
                    split_entities.append((word, tag))
            else:
                split_entities.append((word, tag))
        return split_entities

    def preprocess_pod_names(self, text):
        pod_name_pattern = re.compile(r'\b(\w+(?:-\w+)+)\s+pod\b', re.IGNORECASE)
        matches = pod_name_pattern.findall(text)
        for match in matches:
            text = text.replace(match + " pod", "")
        return text, matches

    def preprocess_files_and_paths(self, text):
        files_and_paths = []
        remaining_text = text

        for match in FILE_PATTERN.finditer(text):
            files_and_paths.append((match.group(0), 'FILE'))
            remaining_text = remaining_text.replace(match.group(0), "")

        for match in PATH_PATTERN.finditer(text):
            if match.group(0) not in remaining_text:
                continue
            files_and_paths.append((match.group(0), 'PATH'))
            remaining_text = remaining_text.replace(match.group(0), "")

        return remaining_text, files_and_paths

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--text', type=str, required=True, help="Input text for prediction")
    args = parser.parse_args()

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")

    predictor = Predictor('./pkg/promptprocessor/models/best_entity_model.bin', device)
    
    preprocessed_text, pod_names = predictor.preprocess_pod_names(args.text)
    preprocessed_text, files_and_paths = predictor.preprocess_files_and_paths(preprocessed_text)
    predicted_entities = predictor.predict_entities(preprocessed_text)

    for pod_name in pod_names:
        predicted_entities.append((pod_name, 'POD_NAME'))
    predicted_entities.extend(files_and_paths)


    result = {
        "entities": [{"text": e[0], "type": e[1]} for e in predicted_entities]
    }

    print(json.dumps(result))

if __name__ == "__main__":
    main()
