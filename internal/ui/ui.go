package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	client := &http.Client{}
	url := "http://localhost:9090/intent"

	for {
		fmt.Print("Enter your intent (or type 'q' to quit): ")
		intent, _ := reader.ReadString('\n')
		intent = strings.TrimSpace(intent)

		if intent == "q" {
			fmt.Println("Exiting...")
			break
		}

		req, err := http.NewRequest("POST", url, strings.NewReader(intent))
		if err != nil {
			fmt.Println("Error creating request:", err)
			continue
		}
		req.Header.Set("Content-Type", "text/plain")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Println("Intent Enforcement successfully")
			fmt.Println()
		} else {
			fmt.Println()
			fmt.Printf("Failed to submit intent. Status code: %d\n", resp.StatusCode)
		}
	}
}
