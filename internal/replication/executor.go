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
	"strconv"
	"strings"
	"time"
)

type Executor struct {
	secondaryUrls []string
	// Clients are safe for concurrent use by multiple goroutines. https://go.dev/src/net/http/client.go
	client http.Client
	// retry config
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

	replicationTimeout := 50 * time.Millisecond // default value
	if replicationTimeoutToken, okTimeout := os.LookupEnv("REPLICATION_TIMEOUT_MILLISECONDS"); okTimeout {
		value, _ := strconv.Atoi(replicationTimeoutToken)
		replicationTimeout = time.Duration(value) * time.Millisecond
	}

	initialSleepTime := 10 * time.Millisecond // default value
	intervalValue := int64(initialSleepTime) / 2

	return &Executor{
		secondaryUrls: secondaryUrls,
		client: http.Client{
			Timeout: replicationTimeout,
		},
		// retry config
		initialSleepTime: initialSleepTime,
		sleepMultiplier:  2,
		// jitter config
		maxInterval: intervalValue,
		minInterval: -intervalValue,
	}
}

func (e *Executor) ReplicateMessage(message model.Message, w int) {
	if w > len(e.secondaryUrls) {
		log.Fatalf("w > primaries number, %d > %d", w, len(e.secondaryUrls))
	}

	// Buffered channels allows to accept a limited number of values without a corresponding receiver for those values
	replicationIsFinished := make(chan struct{}, len(e.secondaryUrls))

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

	// WHILE NOT SUCCESS:
	for attempt := 0; ; attempt++ {
		// 1) Send Request
		req := io.NopCloser(strings.NewReader(reqBody))
		log.Printf("Sending message %d to %s. Attempt %d.", message.Id, secondaryUrl, attempt)
		resp, err := e.client.Post(secondaryUrl+"/api/v1/internal/replicate", "application/json", req)

		// 2) Handle Response
		if err != nil {
			log.Printf("Failed to replicate message. Err: %s", err)
		} else if resp.StatusCode != 200 {
			log.Printf("Failed to replicate message. Secondary url: %s, status code: %d", secondaryUrl, resp.StatusCode)
		} else {
			log.Printf("ACK (message %d). Secondary url: %s\n", message.Id, secondaryUrl)
			// SUCCESS! Notify main thread and exit...
			notify <- struct{}{}
			return
		}

		// 3) Sleep in case of Failure or DEAD Secondary
		currentSleepTime := e.calculateCurrentSleepTime(attempt)
		log.Printf("Sleeping %v ms before next retry...", currentSleepTime)
		time.Sleep(currentSleepTime)
	}
}

// wait_interval = (base * multiplier^n) +/- (random interval)
func (e *Executor) calculateCurrentSleepTime(failures int) time.Duration {
	randomInterval := time.Duration(rand.Int63n(e.maxInterval-e.minInterval) + e.minInterval)
	multiplierPowN := time.Duration(math.Pow(float64(e.sleepMultiplier), float64(failures)))

	waitInterval := (e.initialSleepTime * multiplierPowN) + randomInterval
	return waitInterval
}
