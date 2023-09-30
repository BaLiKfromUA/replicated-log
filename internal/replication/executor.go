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
	"sync"
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
	if len(secondaryUrls) == 0 {
		log.Fatalf("Given 'SECONDARY_URLS' token is empty")
	}

	isValid := true
	for _, secondaryUrl := range secondaryUrls {
		isValid = isValidUrl(secondaryUrl) && isValid
	}
	if !isValid {
		log.Fatalf("Given 'SECONDARY_URLS' token is invalid: '%s'", secondaryUrlsToken)

	}

	return &Executor{secondaryUrls: secondaryUrls, client: http.Client{}}
}

func (e *Executor) ReplicateMessage(message model.Message) {
	payload, _ := json.Marshal(message)
	reqBody := string(payload)

	var wg sync.WaitGroup
	wg.Add(len(e.secondaryUrls))

	log.Printf("Replicating message %d\n", message.Id)
	for _, secondaryUrl := range e.secondaryUrls {
		go func(url, reqBody string) {
			defer wg.Done()

			req := io.NopCloser(strings.NewReader(reqBody))
			resp, err := e.client.Post(url+"/api/v1/replicate", "application/json", req)

			if err != nil {
				// at this stage assume that the communication channel is a perfect link (no failures and messages lost)
				log.Fatalf("Failed to replicate message. Secondary url: %s, err: %s", url, err)
			} else if resp.StatusCode != 200 {
				// at this stage assume that the communication channel is a perfect link (no failures and messages lost)
				log.Fatalf("Failed to replicate message. Secondary url: %s, status code: %d", url, resp.StatusCode)
			} else {
				log.Printf("ACK (message %d). Secondary url: %s\n", message.Id, url)
			}
		}(secondaryUrl, reqBody)
	}

	wg.Wait()
}
