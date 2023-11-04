package replication

import (
	"encoding/json"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"replicated-log/internal/model"
	"strings"
	"time"
)

type Executor struct {
	secondaryUrls []string
	// Clients are safe for concurrent use by multiple goroutines. https://go.dev/src/net/http/client.go
	client           http.Client
	initialSleepTime time.Duration
	sleepMultiplier  int
	// jitter config
	minInterval int64
	maxInterval int64
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

	// todo: set flexible with env variable

	initialSleepTime := 10 * time.Millisecond
	intervalValue := int64(initialSleepTime) / 2

	return &Executor{
		secondaryUrls: secondaryUrls,
		client: http.Client{
			Timeout: 1 * time.Second,
		},
		initialSleepTime: initialSleepTime,
		sleepMultiplier:  2,
		maxInterval:      intervalValue,
		minInterval:      -intervalValue,
	}
}

func (e *Executor) ReplicateMessage(message model.Message, w int) {
	if w > len(e.secondaryUrls) {
		log.Fatalf("w > primaries number, %d > %d", w, len(e.secondaryUrls))
	}

	// Buffered channels allows to accept a limited number of values without a corresponding receiver for those values
	replicationIsFinished := make(chan struct{}, len(e.secondaryUrls))

	log.Printf("Replicating message %d\n", message.Id)
	for _, secondaryUrl := range e.secondaryUrls {
		go e.replicateWithRetry(secondaryUrl, message, replicationIsFinished)
	}

	for w > 0 {
		<-replicationIsFinished
		w--
	}
}

func (e *Executor) replicateWithRetry(secondaryUrl string, message model.Message, notify chan<- struct{}) {
	payload, _ := json.Marshal(message)
	reqBody := string(payload)

	failures := 0

	var currentSleepTime time.Duration
	for {
		randomInterval := time.Duration(rand.Int63n(e.maxInterval-e.minInterval) + e.minInterval)
		multiplierPowN := time.Duration(math.Pow(float64(e.sleepMultiplier), float64(failures)))
		// wait_interval = (base * multiplier^n) +/- (random interval)
		currentSleepTime = (e.initialSleepTime * multiplierPowN) + randomInterval

		req := io.NopCloser(strings.NewReader(reqBody))
		log.Printf("Sending message %d to %s. Attempt %d.", message.Id, secondaryUrl, failures)
		resp, err := e.client.Post(secondaryUrl+"/api/v1/internal/replicate", "application/json", req)

		if err != nil {
			log.Printf("Failed to replicate message. Secondary url: %s, err: %s", secondaryUrl, err)
		} else if resp.StatusCode != 200 {
			log.Printf("Failed to replicate message. Secondary url: %s, status code: %d", secondaryUrl, resp.StatusCode)
		} else {
			log.Printf("ACK (message %d). Secondary url: %s\n", message.Id, secondaryUrl)
			notify <- struct{}{}
			return
		}
		failures += 1

		log.Printf("Sleeping %v ms before next retry...", currentSleepTime)
		time.Sleep(currentSleepTime)
	}
}
