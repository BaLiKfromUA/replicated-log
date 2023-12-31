package replication

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"replicated-log/internal/model"
	"sync"
	"testing"
	"time"
)

func TestReplicateMessageWithOneSecondary(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "first one"}

	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// health-check
			rw.WriteHeader(http.StatusOK)
		} else {
			var actualMessage model.Message
			err := json.NewDecoder(r.Body).Decode(&actualMessage)
			// THEN
			require.NoError(t, err)
			require.Equal(t, message, actualMessage)
			rw.WriteHeader(http.StatusOK)
		}
	}))
	defer secondary.Close()

	t.Setenv("SECONDARY_URLS", secondary.URL)

	// WHEN
	NewExecutor().ReplicateMessage(message, 1)
}

func TestReplicateMessageWithTwoSecondaries(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "first one"}

	handler := func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// health-check
			rw.WriteHeader(http.StatusOK)
		} else {
			var actualMessage model.Message
			err := json.NewDecoder(r.Body).Decode(&actualMessage)
			// THEN
			require.NoError(t, err)
			require.Equal(t, message, actualMessage)
			rw.WriteHeader(http.StatusOK)
		}
	}

	secondaryA := httptest.NewServer(http.HandlerFunc(handler))
	defer secondaryA.Close()
	secondaryB := httptest.NewServer(http.HandlerFunc(handler))
	defer secondaryB.Close()

	t.Setenv("SECONDARY_URLS", secondaryA.URL+","+secondaryB.URL)

	// WHEN
	NewExecutor().ReplicateMessage(message, 2)
}

func TestReplicateMessageWithTwoSecondariesDelayedResponse(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "first one"}
	ready := make(chan struct{}, 2) // to emulate delay

	handler := func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// health-check
			rw.WriteHeader(http.StatusOK)
		} else {
			<-ready // artificial delay
			var actualMessage model.Message
			err := json.NewDecoder(r.Body).Decode(&actualMessage)
			// THEN
			require.NoError(t, err)
			require.Equal(t, message, actualMessage)
			rw.WriteHeader(http.StatusOK)
		}
	}

	secondaryA := httptest.NewServer(http.HandlerFunc(handler))
	defer secondaryA.Close()
	secondaryB := httptest.NewServer(http.HandlerFunc(handler))
	defer secondaryB.Close()

	t.Setenv("SECONDARY_URLS", secondaryA.URL+","+secondaryB.URL)

	// WHEN
	ready <- struct{}{} // unblock 1 secondary server
	// one secondary should block replication, but we need only 1 ACK
	NewExecutor().ReplicateMessage(message, 1)
	ready <- struct{}{} // unblock all
}

func TestReplicateWithRetrySendsSameRequestEveryTimeAndNotifiesAboutSuccess(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "first one"}
	maxTrials := 3
	currentTrial := 0

	handler := func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// health-check
			rw.WriteHeader(http.StatusOK)
		} else {
			var actualMessage model.Message
			err := json.NewDecoder(r.Body).Decode(&actualMessage)
			// THEN
			require.NoError(t, err)
			require.Equal(t, message, actualMessage)

			if currentTrial >= maxTrials {
				rw.WriteHeader(http.StatusOK)
			} else {
				rw.WriteHeader(http.StatusRequestTimeout)
				currentTrial++
			}
		}
	}

	secondary := httptest.NewServer(http.HandlerFunc(handler))
	defer secondary.Close()

	// just for successful initialization, doesn't play role in this test:
	t.Setenv("SECONDARY_URLS", secondary.URL)

	success := make(chan struct{}, 1)

	// WHEN
	NewExecutor().replicateWithRetry(secondary.URL, message, success)

	// THEN
	<-success // block till notification
	require.Equal(t, currentTrial, maxTrials)
}

func TestReplicateWithRetryWorksCorrectWithClientTimeout(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "first one"}
	maxTrials := 3
	currentTrial := 0
	var mu sync.Mutex

	handler := func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// health-check
			rw.WriteHeader(http.StatusOK)
		} else {
			var actualMessage model.Message
			err := json.NewDecoder(r.Body).Decode(&actualMessage)
			// THEN
			require.NoError(t, err)
			require.Equal(t, message, actualMessage)

			mu.Lock()
			if currentTrial < maxTrials {
				// !!! Sleep time is much bigger than request timeout
				currentTrial++
				mu.Unlock()
				time.Sleep(50 * time.Millisecond)
			} else {
				mu.Unlock()
			}
			rw.WriteHeader(http.StatusOK)
		}
	}

	secondary := httptest.NewServer(http.HandlerFunc(handler))
	defer secondary.Close()

	// just for successful initialization, doesn't play role in this test:
	t.Setenv("SECONDARY_URLS", secondary.URL)
	// Client timeout is very small
	t.Setenv("REQUEST_TIMEOUT_MILLISECONDS", "10")

	success := make(chan struct{}, 1)

	// WHEN
	NewExecutor().replicateWithRetry(secondary.URL, message, success)

	// THEN
	<-success // block till notification
	require.Equal(t, currentTrial, maxTrials)
}
