import torch
import torch.nn as nn
from transformers import BertModel, BertTokenizerFast
import time
import os
from datetime import datetime
from data import DataProcessor
from model import BertForIntentClassification, BertForEntityClassification
from custom_tokenizer import CustomBertTokenizer
from train_eval import Trainer
from test import Tester
from plot import save_training_results

os.environ["CUDA_VISIBLE_DEVICES"] = "1"
index_to_entity_tag = {0: 'POLICY', 1: 'LABEL', 2: 'POD_NAME', 3: 'NAMESPACE', 4: 'ACTION',
                       5: 'TRAFFIC_DIRECTION', 6: 'CIDR', 7: 'PORT', 8: 'PROTOCOL',
                       9: 'PATH', 10: 'FILE', 11: 'HTTP_PATH', 12: 'HTTP_METHOD', 
                       13: 'FQDN', 14: 'ENDPOINT', 15: 'TRASH'}

def train_and_evaluate(entity_lr, entity_num_epochs, entity_batch_size):
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    
    data_processor = DataProcessor(file_path='./dataset/intent_v5.csv')
    df = data_processor.load_and_process_data()
    train_dataset, valid_dataset, test_dataset = data_processor.create_datasets(df)
    train_loader, valid_loader, test_loader = data_processor.create_dataloaders(train_dataset, valid_dataset, test_dataset, max(entity_batch_size))

    bert_model = BertModel.from_pretrained('bert-base-uncased')
    encode_plus_custom = CustomBertTokenizer.from_pretrained('bert-base-uncased', use_fast=True)

    entity_model = BertForEntityClassification(n_entity_types=len(index_to_entity_tag), bert_model=bert_model)
    entity_model.to(device)
    entity_optimizer = torch.optim.Adam(entity_model.parameters(), entity_lr)
    entity_criterion = nn.CrossEntropyLoss()

    entity_trainer = Trainer(entity_model, entity_optimizer, entity_criterion, device, custom_tokenizer=encode_plus_custom)

    best_entity_accuracy = 0

    train_losses = []
    val_losses = []
    train_entity_accs = []
    val_entity_accs = []
    total_start_time = time.time()
    
    for epoch in range(max(entity_num_epochs)):
        epoch_start_time = time.time()
        current_time = datetime.now().strftime("%H:%M")
        print(f'{current_time} Epoch {epoch + 1}/{max(entity_num_epochs)}')

        if epoch < entity_num_epochs:
            # 엔티티 분류 모델 학습 및 검증
            train_entity_loss, train_entity_acc = entity_trainer.train_entity(train_loader)
            val_entity_loss, val_entity_acc = entity_trainer.eval_entity(valid_loader)

            train_entity_accs.append(train_entity_acc)
            val_entity_accs.append(val_entity_acc)

            print(f'Entity - Train Loss: {train_entity_loss:.4f}, Train Acc: {train_entity_acc:.4f}')
            print(f'Entity - Val Loss: {val_entity_loss:.4f}, Val Acc: {val_entity_acc:.4f}')

            if val_entity_acc > best_entity_accuracy:
                torch.save(entity_model.state_dict(), 'best_entity_model.bin')
                best_entity_accuracy = val_entity_acc
        
    total_end_time = time.time()
    total_duration = total_end_time - total_start_time
    total_duration_minutes = total_duration / 60
    print(f'Training complete. Total duration: {total_duration_minutes:.2f} minutes')
    save_training_results(train_losses, val_losses, train_entity_accs, val_entity_accs, "./dataset/training_results.csv")
    return test_dataset, test_loader, device


def test_models(test_loader, device):
    bert_model = BertModel.from_pretrained('bert-base-uncased')
    entity_model = BertForEntityClassification(n_entity_types=len(index_to_entity_tag), bert_model=bert_model)
    
    tester = Tester(entity_model, device)
    
    entity_acc, avg_entity_loss, total_loss = tester.test('best_entity_model.bin', test_loader)
    
    print(f'Test Loss: {total_loss:.4f}')
    print(f'\tEntity Loss: {avg_entity_loss:.4f}')
    print(f'\tEntity Acc: {entity_acc:.4f}')


if __name__ == "__main__":
    entity_lr = 1e-3
    entity_num_epochs = 35
    entity_batch_size = 32

    test_dataset, test_loader, device = train_and_evaluate(entity_lr, entity_num_epochs, entity_batch_size)
    test_models(test_loader, device)
