package replication

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"replicated-log/internal/model"
	"strings"
)

type Executor struct {
	secondaryUrls []string
	// Clients are safe for concurrent use by multiple goroutines. https://go.dev/src/net/http/client.go
	client http.Client
}

// isValidUrl tests a string to determine if it is a well-structured url or not.
func isValidUrl(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		log.Printf("'%s' is an invalid URL", toTest)
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Printf("'%s' is an invalid URL", toTest)
		return false
	}

	return true
}

func NewExecutor() *Executor {
	secondaryUrlsToken, ok := os.LookupEnv("SECONDARY_URLS")
	if !ok {
		log.Fatalf("'SECONDARY_URLS' env var is not set")
	}

	secondaryUrls := strings.Split(secondaryUrlsToken, ",")
	isValid := true
	for _, secondaryUrl := range secondaryUrls {
		isValid = isValidUrl(secondaryUrl) && isValid
	}

	if len(secondaryUrls) == 0 || !isValid {
		log.Fatalf("Given 'SECONDARY_URLS' token is invalid: '%s'", secondaryUrlsToken)
	}

	return &Executor{secondaryUrls: secondaryUrls, client: http.Client{}}
}

func (e *Executor) ReplicateMessage(message model.Message) {
	payload, _ := json.Marshal(message)
	reqBody := string(payload)

	for _, secondaryUrl := range e.secondaryUrls {
		req := io.NopCloser(strings.NewReader(reqBody))
		resp, err := e.client.Post(secondaryUrl+"/api/v1/replicate", "application/json", req)

		if err != nil || resp.StatusCode != http.StatusOK {
			log.Printf("Failed to replicate message. Secondary url: %s, err: %s, status code: %d", secondaryUrl, err, resp.StatusCode)
		} else {
			log.Printf("ACK. Secondary url: %s", secondaryUrl)
		}
	}
}
