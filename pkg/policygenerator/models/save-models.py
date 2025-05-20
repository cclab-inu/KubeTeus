from transformers import AutoModelForCausalLM, AutoTokenizer

def download_and_save_model(model_name, save_directory):
    model = AutoModelForCausalLM.from_pretrained(model_name)
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    
    model.save_pretrained(save_directory)
    tokenizer.save_pretrained(save_directory)

if __name__ == "__main__":
    # deepseek_base_system_v2
    #model_name = "cclabadmin/deepseek_base_system_v2"  # Model name to use
    #save_directory = "./pkg/executor/models/deepseek_base_system_v2"  # Directory path to save
    
    #model_name = "cclabadmin/codegemma-7b-it-network"  # Model name to use
    #save_directory = "./pkg/executor/models/codegemma-7b-it-network"  # Directory path to save
    
    model_name = "cclabadmin/deepseek_base_network"
    save_directory = "./pkg/executor/models/deepseek_base_network"
    download_and_save_model(model_name, save_directory)
