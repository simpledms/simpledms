package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"

	"golang.org/x/term"
)

// usage: `unlocker https://app.simpledms.ch/-/unlock-cmd`
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

	resp, err := sendUnlockRequest(url, passphrase)
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

func sendUnlockRequest(url, passphrase string) (*http.Response, error) {
	jsonData, err := json.Marshal(struct {
		Passphrase string `json:"passphrase"`
	}{
		Passphrase: passphrase,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return http.DefaultClient.Do(req)
}
