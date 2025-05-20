import re
from transformers import BertTokenizerFast

class CustomBertTokenizer:
    def __init__(self, tokenizer):
        self.tokenizer = tokenizer

    @classmethod
    def from_pretrained(cls, pretrained_model_name_or_path, *init_inputs, **kwargs):
        tokenizer = BertTokenizerFast.from_pretrained(pretrained_model_name_or_path, *init_inputs, **kwargs)
        custom_tokenizer = cls(tokenizer)  
        return custom_tokenizer

    def custom_tokenize(self, text):
        patterns = {
            "POLICY": re.compile(r'\b(CiliumNetworkPolicy|network policy|security policy|KubeArmorHostPolicy|KubeArmorPolicy|policy|Policy)\b', re.IGNORECASE),
            "LABEL": re.compile(r"([\w\.-]+)\s*:\s*([\w\.-]+)|(matchLabels|matchlabels)"),
            "POD_NAME": re.compile(r"'\s*([^']+?)\s*'\s+(pod|pods)\b|\b([^'\s]+)\s+(pod|pods)\b", re.IGNORECASE),
            "NAMESPACE": re.compile(r"(?<!\w\s)(?<!['\w])\b(\w+)\s+(namespace|namespaces)\b|'([^']+)'(?:\s+namespace)?\b|namespace\s+'([^']+)'", re.IGNORECASE),
            "ACTION": re.compile(r'\b(allow|deny|block|audit|restrict|prevent|denies|prevents|blocking)\b', re.IGNORECASE),
            "ENDPOINT": re.compile(r'\b(endpoints|endpoint)\b', re.IGNORECASE),
            "TRAFFIC_DIRECTION": re.compile(r'\b(ingress|egress|income|incoming|inbound|outcome|outcoming|outbound)\b', re.IGNORECASE),
            "CIDR": re.compile(r'\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/\d{1,2})\b'),
            "PORT": re.compile(r'\bport (\d+)\b|\b(\d+) port\b|\bport\b|\'(\d+)\'', re.IGNORECASE),
            "PROTOCOL": re.compile(r'\b(TCP|UDP|ICMP)\b', re.IGNORECASE),
            "PATH": re.compile(r'(/[a-zA-Z_][-a-zA-Z0-9_/.]*[^/])'),
            "FILE": re.compile(r'(/[\w/\.-]+\.\w+)(?![/\w])'),
            "HTTP_METHOD": re.compile(r'\b(POST|GET)\b', re.IGNORECASE),
            "HTTP_PATH": re.compile(r'\b(/[a-zA-Z][a-zA-Z0-9_/.-]*(?![\d/]))\b'),
            "FQDN": re.compile(r'\b([\w-]+\.)+[\w-]+\.[a-z]{2,}\b(?!:\s*)')
        }

        matches = []
        for entity_type, pattern in patterns.items():
            for match in pattern.finditer(text):
                matches.append((match.start(), match.end(), match.group(0), entity_type))

        matches.sort()
        merged_tokens = []
        last_end = 0
        for start, end, token, entity_type in matches:
            if start >= last_end:
                merged_tokens.append({"text": token, "start": start, "end": end, "type": entity_type})
                last_end = end

        return merged_tokens

    def encode_plus_custom(self, text, add_special_tokens=True, max_length=None, padding='max_length', truncation=True, return_tensors="pt"):
        custom_tokens = self.custom_tokenize(text)
        custom_text = " ".join([token["text"] for token in custom_tokens])
        return self.tokenizer.encode_plus(
            custom_text,
            add_special_tokens=add_special_tokens,
            max_length=max_length,
            padding=padding,
            truncation=truncation,
            return_tensors=return_tensors
        )

    def convert_ids_to_tokens(self, input_ids):
        return self.tokenizer.convert_ids_to_tokens(input_ids)

#tokenizer = CustomBertTokenizer.from_pretrained('bert-base-uncased')
#text = "Create a policy in the nginx pod to deny egress traffic coming in on port 80 TCP From endpoints labeled 'app: nginx'"
#tokens = tokenizer.custom_tokenize(text)
#for token in tokens:
#    print(token)
