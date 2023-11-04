package replication

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

type HealthCheckMonitoringDaemon struct {
	mu                *sync.Mutex
	secondaryUrls     []string
	secondaryStatuses map[string]string
	client            http.Client
	quit              chan struct{}
}

func NewHealthCheckMonitoringDaemon(urls []string) *HealthCheckMonitoringDaemon {
	replicationTimeout := 50 * time.Millisecond // default value
	if replicationTimeoutToken, okTimeout := os.LookupEnv("REPLICATION_TIMEOUT_MILLISECONDS"); okTimeout {
		value, _ := strconv.Atoi(replicationTimeoutToken)
		replicationTimeout = time.Duration(value) * time.Millisecond
	}

	daemon := HealthCheckMonitoringDaemon{
		mu:                &sync.Mutex{},
		secondaryUrls:     urls,
		secondaryStatuses: make(map[string]string),
		client: http.Client{
			Timeout: replicationTimeout,
		},
		quit: make(chan struct{}, 1),
	}

	daemon.doHealthCheck(true)

	return &daemon
}

func (daemon *HealthCheckMonitoringDaemon) StartHealthCheck() {
	log.Printf("[HEALTH-CHECK] START health check background thread")
	ticker := time.NewTicker(100 * time.Millisecond)
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

func (daemon *HealthCheckMonitoringDaemon) StopHealthCheck() {
	log.Printf("[HEALTH-CHECK] FINISH health check background thread")
	close(daemon.quit)
}

func (daemon *HealthCheckMonitoringDaemon) doHealthCheck(isInit bool) {
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

func (daemon *HealthCheckMonitoringDaemon) checkHealth(secondaryUrl string) {
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

func (daemon *HealthCheckMonitoringDaemon) GetStatus(url string) string {
	daemon.mu.Lock()
	defer daemon.mu.Unlock()

	status, ok := daemon.secondaryStatuses[url]

	if !ok {
		log.Fatalf("[HEALTH-CHECK] %s is not found!", url)
	}

	return status
}
