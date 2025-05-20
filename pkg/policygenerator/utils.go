package policygenerator

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cclab-inu/KubeTeus/pkg/utils"
	"gopkg.in/yaml.v3"
)

func saveOutputToFile(filename, output string) {
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer f.Close()
	_, err = f.WriteString(output)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
	}
}

func filterOutput(output string) string {
	lines := strings.Split(output, "\n")
	var filteredLines []string
	for _, line := range lines {
		if !strings.Contains(line, "Requirement already satisfied") &&
			!strings.Contains(line, "Loading checkpoint shards") {
			filteredLines = append(filteredLines, line)
		}
	}
	return strings.Join(filteredLines, "\n")
}

func logFailureWithTimestamp(message string) {
	fmt.Printf("%s: %s", time.Now().Format(time.RFC3339), message)
}

func readConfig(path string) (*utils.Configuration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config utils.Configuration
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
