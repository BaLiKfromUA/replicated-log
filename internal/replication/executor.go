package replication

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
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

func NewExecutor() *Executor {
	secondaryUrlsToken, ok := os.LookupEnv("SECONDARY_URLS")
	if !ok {
		log.Fatalf("'SECONDARY_URLS' env var is not set")
	}

	secondaryUrls := strings.Split(secondaryUrlsToken, ",")

	if len(secondaryUrls) == 0 {
		log.Fatalf("Given 'SECONDARY_URLS' token is empty")
	}

	return &Executor{secondaryUrls: secondaryUrls, client: http.Client{}}
}

func (e *Executor) ReplicateMessage(message model.Message) {
	payload, _ := json.Marshal(message)
	reqBody := string(payload)

	var wg sync.WaitGroup
	wg.Add(len(e.secondaryUrls))

	for _, secondaryUrl := range e.secondaryUrls {
		go func(url, reqBody string) {
			defer wg.Done()

			req := io.NopCloser(strings.NewReader(reqBody))
			resp, err := e.client.Post(url+"/api/v1/replicate", "application/json", req)

			if err != nil || resp.StatusCode != http.StatusOK {
				log.Printf("Failed to replicate message. Secondary url: %s, err: %s, status code: %d", url, err, resp.StatusCode)
			} else {
				log.Printf("ACK. Secondary url: %s", url)
			}
		}(secondaryUrl, reqBody)
	}

	wg.Wait()
}
