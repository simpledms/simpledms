package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"

	"golang.org/x/term"
)

// usage: `unlocker https://app.simpledms.ch/-/cmd/unlock`
func main() {
	// Get the first parameter passed to the command and save it in the variable "url"
	if len(os.Args) < 2 {
		log.Fatalln("Usage: go run main.go <url>")
	}
	url := os.Args[1]

	fmt.Println("Enter passphrase: ")
	passphraseBytes, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		log.Println("error reading passphrase:", err)
		return
	}

	passphrase := string(passphraseBytes)
	fmt.Println()
	fmt.Println("Unlocking...")
	fmt.Println()

	jsonData := []byte(fmt.Sprintf(`{"passphrase": "%s"}`, passphrase))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Println("error, status code was", resp.StatusCode)

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Println("error reading response body:", readErr)
			return
		}
		log.Println(string(body))

		return
	}

	fmt.Println("Successfully unlocked!")
}
