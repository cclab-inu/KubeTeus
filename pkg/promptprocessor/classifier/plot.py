import matplotlib.pyplot as plt
import pandas as pd


def plot_train_res(train_losses, val_losses, train_intent_accs, val_intent_accs, train_entity_accs, val_entity_accs, save_path=None):
    plt.figure(figsize=(12, 6))

    plt.subplot(1, 2, 1)
    plt.plot(train_losses, label='Train Loss')
    plt.plot(val_losses, label='Validation Loss')
    plt.title('Loss over epochs')
    plt.xlabel('Epoch')
    plt.ylabel('Loss')
    plt.legend()

    plt.subplot(1, 2, 2)
    plt.plot(train_intent_accs, label='Train Intent Accuracy')
    plt.plot(val_intent_accs, label='Validation Intent Accuracy')
    plt.plot(train_entity_accs, label='Train Entity Accuracy')
    plt.plot(val_entity_accs, label='Validation Entity Accuracy')
    plt.title('Accuracy over epochs')
    plt.xlabel('Epoch')
    plt.ylabel('Accuracy')
    plt.legend()

    if save_path:
        plt.savefig(save_path)
    else:
        plt.show()


def save_training_results(train_losses, val_losses, train_intent_accs, val_intent_accs, train_entity_accs, val_entity_accs, filename='training_results.csv'):
    data = {
        'epoch': list(range(1, len(train_losses) + 1)),
        'train_loss': train_losses,
        'val_loss': val_losses,
        'train_intent_acc': train_intent_accs,
        'val_intent_acc': val_intent_accs,
        'train_entity_acc': train_entity_accs,
        'val_entity_acc': val_entity_accs,
    }
    df = pd.DataFrame(data)
    df.to_csv(filename, index=False)


def main():
    csv_file = './dataset/training_results.csv'
    df = pd.read_csv(csv_file)
    
    train_losses = df['train_loss'].tolist()
    val_losses = df['val_loss'].tolist()
    train_intent_accs = df['train_intent_acc'].tolist()
    val_intent_accs = df['val_intent_acc'].tolist()
    train_entity_accs = df['train_entity_acc'].tolist()
    val_entity_accs = df['val_entity_acc'].tolist()

    plot_train_res(train_losses, val_losses, train_intent_accs, val_intent_accs, train_entity_accs, val_entity_accs, save_path='./dataset/training_results.png')

if __name__ == "__main__":
    main()