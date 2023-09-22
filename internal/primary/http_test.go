package primary

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"replicated-log/internal/model"
	"replicated-log/internal/storage"
	"strings"
	"sync"
	"testing"
)

func TestAppendMessageWithOneSecondary(t *testing.T) {
	// GIVEN
	messageRequest := AppendMessageRequest{W: 2, Message: "Test"}
	b, _ := json.Marshal(messageRequest)

	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var actualMessage model.Message
		err := json.NewDecoder(r.Body).Decode(&actualMessage)
		// THEN
		assert.NoError(t, err)
		assert.Equal(t, messageRequest.Message, actualMessage.Message)
		rw.WriteHeader(http.StatusOK)
	}))
	defer secondary.Close()

	t.Setenv("SECONDARY_URLS", secondary.URL)
	primary := NewPrimaryServer()
	handler := primary.Handler

	t.Run("Initial message list is empty", func(t *testing.T) {
		// GIVEN
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "{\"messages\":[]}", string(body))
	})

	t.Run("Append a new message", func(t *testing.T) {
		// GIVEN
		req := httptest.NewRequest(http.MethodPost, "/api/v1/append", strings.NewReader(string(b)))
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Update message list contains appended message", func(t *testing.T) {
		// GIVEN
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "{\"messages\":[\"Test\"]}", string(body))
	})
}

func TestReplicationWithEmulatedDelayForOneSecondaryOutOfTwo(t *testing.T) {
	ready := make(chan struct{}, 1)   // to emulate delay
	updated := make(chan struct{}, 1) // to emulate request/response to secondary

	storageA := storage.NewInMemoryStorage()
	secondaryA := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var message model.Message
		err := json.NewDecoder(r.Body).Decode(&message)
		require.NoError(t, err)
		_ = storageA.AddMessage(message)
		rw.WriteHeader(http.StatusOK)
	}))
	defer secondaryA.Close()

	storageB := storage.NewInMemoryStorage()
	secondaryB := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		<-ready // artificial delay
		var message model.Message
		err := json.NewDecoder(r.Body).Decode(&message)
		require.NoError(t, err)
		_ = storageB.AddMessage(message)
		updated <- struct{}{} // notify test
		rw.WriteHeader(http.StatusOK)
	}))
	defer secondaryB.Close()

	t.Setenv("SECONDARY_URLS", secondaryA.URL+","+secondaryB.URL)
	primary := NewPrimaryServer()
	handler := primary.Handler

	t.Run("Append a new message but one secondary is blocked", func(t *testing.T) {
		// GIVEN
		w := 2 // ACK from master and one secondary
		messageRequest := AppendMessageRequest{W: w, Message: "Test"}
		b, _ := json.Marshal(messageRequest)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/append", strings.NewReader(string(b)))
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Primary and Secondary A have the same message", func(t *testing.T) {
		// GIVEN
		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		var data GetMessagesResponse
		err = json.Unmarshal(body, &data)
		assert.NoError(t, err)

		assert.Equal(t, data.Messages, storageA.GetMessages())
	})

	t.Run("Secondary B doesn't have any messages", func(t *testing.T) {
		// THEN
		assert.Empty(t, storageB.GetMessages())
	})

	t.Run("After fixing the delay, Secondary A and Secondary B have the same messages", func(t *testing.T) {
		// WHEN
		ready <- struct{}{} // unblock secondary
		// THEN
		<-updated // wait for update
		assert.Equal(t, storageA.GetMessages(), storageB.GetMessages())
	})
}

func TestReplicationWithSecondariesBothBlocked(t *testing.T) {
	var wgForStorageUpdates sync.WaitGroup
	wgForStorageUpdates.Add(2)
	ready := make(chan struct{}, 2) // to emulate delay

	// GIVEN
	w := 1 // ACK only from master
	expectedMessage := "Test"
	secondaryHandler := func(rw http.ResponseWriter, r *http.Request) {
		<-ready // artificial delay
		var actualMessage model.Message
		err := json.NewDecoder(r.Body).Decode(&actualMessage)
		// THEN
		require.NoError(t, err)
		require.Equal(t, expectedMessage, actualMessage.Message)
		rw.WriteHeader(http.StatusOK)
		wgForStorageUpdates.Done()
	}

	secondaryA := httptest.NewServer(http.HandlerFunc(secondaryHandler))
	defer secondaryA.Close()
	secondaryB := httptest.NewServer(http.HandlerFunc(secondaryHandler))
	defer secondaryB.Close()

	t.Setenv("SECONDARY_URLS", secondaryA.URL+","+secondaryB.URL)
	primary := NewPrimaryServer()
	handler := primary.Handler

	messageRequest := AppendMessageRequest{W: w, Message: expectedMessage}
	b, _ := json.Marshal(messageRequest)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/append", strings.NewReader(string(b)))
	resp := httptest.NewRecorder()

	// WHEN
	handler.ServeHTTP(resp, req)

	// THEN
	assert.Equal(t, http.StatusOK, resp.Code)

	ready <- struct{}{}
	ready <- struct{}{}
	close(ready)

	wgForStorageUpdates.Wait() // wait for unblocking of all secondaries
}
