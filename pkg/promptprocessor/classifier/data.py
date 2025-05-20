import pandas as pd # type: ignore
import json
from transformers import BertTokenizerFast
from torch.utils.data import Dataset, DataLoader
import torch
from sklearn.model_selection import train_test_split

class IntentAndNerDataset(Dataset):
    def __init__(self, texts, intents, entities, tokenizer, max_len):
        self.texts = texts
        self.intents = intents
        self.entities = entities
        self.tokenizer = tokenizer
        self.max_len = max_len

    def __len__(self):
        return len(self.texts)

    def __getitem__(self, item):
        text = self.texts[item]
        intent = self.intents[item]
        entity_info = self.entities[item]

        encoding = self.tokenizer(
            text,
            add_special_tokens=True,
            max_length=self.max_len,
            return_token_type_ids=False,
            padding='max_length',
            truncation=True,
            return_attention_mask=True
        )

        labels = [0] * self.max_len
        entities = json.loads(entity_info)
        for ent in entities:
            start = ent['start']
            end = ent['end']
            ent_type = ent['type']

            token_start_index = encoding.char_to_token(start)
            token_end_index = encoding.char_to_token(end - 1)

            if token_start_index is not None and token_end_index is not None:
                for idx in range(token_start_index, token_end_index + 1):
                    labels[idx] = ent_type

        return {
            'input_ids': torch.tensor(encoding['input_ids'], dtype=torch.long),
            'attention_mask': torch.tensor(encoding['attention_mask'], dtype=torch.long),
            'intents': torch.tensor(intent, dtype=torch.long),
            'entities': torch.tensor(labels, dtype=torch.long)
        }

class DataProcessor:
    def __init__(self, file_path, max_len=374):
        self.file_path = file_path
        self.max_len = max_len
        self.tokenizer = BertTokenizerFast.from_pretrained('bert-base-uncased', use_fast=True)

    def load_and_process_data(self):
        df = pd.read_csv(self.file_path)
        df['entities'] = df['entities'].apply(self.convert_quotes)
        return df

    def convert_quotes(self, entities):
        temp_entities = entities.replace('"', '\\"')
        temp_entities = temp_entities.replace("'", '"')
        return temp_entities

    def create_datasets(self, df):
        train_val_texts, test_texts, train_val_intents, test_intents, train_val_entities, test_entities = train_test_split(
            df['text'].values, df['class'].values, df['entities'].values, test_size=0.1, random_state=42)

        train_texts, val_texts, train_intents, val_intents, train_entities, val_entities = train_test_split(
            train_val_texts, train_val_intents, train_val_entities, test_size=0.1, random_state=42)

        train_dataset = IntentAndNerDataset(train_texts, train_intents, train_entities, self.tokenizer, self.max_len)
        valid_dataset = IntentAndNerDataset(val_texts, val_intents, val_entities, self.tokenizer, self.max_len)
        test_dataset = IntentAndNerDataset(test_texts, test_intents, test_entities, self.tokenizer, self.max_len)
        return train_dataset, valid_dataset, test_dataset

    def create_dataloaders(self, train_dataset, valid_dataset, test_dataset, batch_size=32):
        train_loader = DataLoader(train_dataset, batch_size=batch_size, shuffle=True)
        valid_loader = DataLoader(valid_dataset, batch_size=batch_size, shuffle=False)
        test_loader = DataLoader(test_dataset, batch_size=batch_size, shuffle=False)
        return train_loader, valid_loader, test_loader
