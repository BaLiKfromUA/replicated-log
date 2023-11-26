package healthcheck

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	ALIVE = "ALIVE"
	DEAD  = "DEAD"
)

type MonitoringDaemon struct {
	mu                *sync.Mutex
	secondaryUrls     []string
	secondaryStatuses map[string]string
	client            http.Client
	quit              chan struct{}
}

func NewMonitoringDaemon(urls []string) *MonitoringDaemon {
	requestTimeout := 50 * time.Millisecond // default value
	if requestTimeoutToken, okTimeout := os.LookupEnv("REQUEST_TIMEOUT_MILLISECONDS"); okTimeout {
		value, _ := strconv.Atoi(requestTimeoutToken)
		requestTimeout = time.Duration(value) * time.Millisecond
	}

	daemon := MonitoringDaemon{
		mu:                &sync.Mutex{},
		secondaryUrls:     urls,
		secondaryStatuses: make(map[string]string),
		client: http.Client{
			Timeout: requestTimeout,
		},
		quit: make(chan struct{}, 1),
	}

	daemon.doHealthCheck(true)

	return &daemon
}

func (daemon *MonitoringDaemon) StartHealthCheck() {
	log.Printf("[HEALTH-CHECK] START health check background thread")

	period := 500 * time.Millisecond // default value
	if periodToken, okTimeout := os.LookupEnv("HEALTHCHECK_PERIOD_MILLISECOND"); okTimeout {
		value, _ := strconv.Atoi(periodToken)
		period = time.Duration(value) * time.Millisecond
	}

	ticker := time.NewTicker(period)
	go func() {
		for {
			select {
			case <-ticker.C:
				daemon.doHealthCheck(false)
			case <-daemon.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (daemon *MonitoringDaemon) StopHealthCheck() {
	log.Printf("[HEALTH-CHECK] FINISH health check background thread")
	close(daemon.quit)
}

func (daemon *MonitoringDaemon) doHealthCheck(isInit bool) {
	log.Printf("[HEALTH-CHECK] Run periodic health check...")

	for _, url := range daemon.secondaryUrls {
		if isInit {
			// first time let's do blocking
			// in order to collect initial state of system
			daemon.checkHealth(url)
		} else {
			go daemon.checkHealth(url)
		}
	}
}

func (daemon *MonitoringDaemon) checkHealth(secondaryUrl string) {
	resp, err := daemon.client.Get(secondaryUrl + "/api/v1/healthcheck")

	daemon.mu.Lock()
	defer daemon.mu.Unlock()
	if err != nil || resp.StatusCode != 200 {
		daemon.secondaryStatuses[secondaryUrl] = DEAD
	} else {
		daemon.secondaryStatuses[secondaryUrl] = ALIVE
	}

	log.Printf("[HEALTH-CHECK] %s status: %s", secondaryUrl, daemon.secondaryStatuses[secondaryUrl])
}

func (daemon *MonitoringDaemon) GetStatus(url string) string {
	daemon.mu.Lock()
	defer daemon.mu.Unlock()

	status, ok := daemon.secondaryStatuses[url]

	if !ok {
		log.Fatalf("[HEALTH-CHECK] %s is not found!", url)
	}

	return status
}

func (daemon *MonitoringDaemon) NoQuorum() bool {
	daemon.mu.Lock()
	defer daemon.mu.Unlock()

	if len(daemon.secondaryStatuses) == 0 {
		log.Fatal("[HEALTH-CHECK] No data about secondaries health\n")
	}

	for _, status := range daemon.secondaryStatuses {
		if status == ALIVE {
			return false
		}
	}

	return true
}
