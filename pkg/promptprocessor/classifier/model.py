import torch
import torch.nn as nn
from transformers import BertModel

class BertForEntityClassification(nn.Module):
    def __init__(self, n_entity_types, bert_model):
        super(BertForEntityClassification, self).__init__()
        self.bert = bert_model
        for param in self.bert.parameters():
            param.requires_grad = False
        self.entity_classifier = nn.Sequential(
            nn.Linear(self.bert.config.hidden_size, 1024),
            nn.Tanh(),
            nn.Linear(1024, 1024),
            nn.Tanh(),
            nn.Linear(1024, n_entity_types),
        )
        nn.init.xavier_uniform_(self.entity_classifier[0].weight)
        nn.init.xavier_uniform_(self.entity_classifier[2].weight)
        nn.init.xavier_uniform_(self.entity_classifier[4].weight)

    def forward(self, input_ids, attention_mask):
        with torch.no_grad():
            outputs = self.bert(input_ids=input_ids, attention_mask=attention_mask)
        sequence_output = outputs[0]
        entity_logits = self.entity_classifier(sequence_output)
        return entity_logits
