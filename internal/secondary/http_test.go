package secondary

import (
	_ "embed"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"replicated-log/internal/model"
	"strings"
	"testing"
)

func TestReplicateAndGetMessages(t *testing.T) {
	secondary := NewSecondaryServer()
	handler := secondary.Handler

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

	t.Run("Replication of a new message", func(t *testing.T) {
		// GIVEN
		message := model.Message{Id: 0, Message: "Test"}
		b, _ := json.Marshal(message)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/replicate", strings.NewReader(string(b)))
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Update message list contains replicated message", func(t *testing.T) {
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

func TestHealthCheck(t *testing.T) {
	secondary := NewSecondaryServer()
	handler := secondary.Handler

	t.Run("Default health is OK", func(t *testing.T) {
		// GIVEN
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthcheck", nil)
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Block replication", func(t *testing.T) {
		// GIVEN
		message := SwitchReplicationModeRequest{ShouldWait: true}
		b, _ := json.Marshal(message)
		req := httptest.NewRequest(http.MethodPost, "/api/test/replication_block", strings.NewReader(string(b)))
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Health during replication block is not OK", func(t *testing.T) {
		// GIVEN
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthcheck", nil)
		resp := httptest.NewRecorder()

		// WHEN

		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusNotAcceptable, resp.Code)
	})

	t.Run("Unblock replication", func(t *testing.T) {
		// GIVEN
		message := SwitchReplicationModeRequest{ShouldWait: false}
		b, _ := json.Marshal(message)
		req := httptest.NewRequest(http.MethodPost, "/api/test/replication_block", strings.NewReader(string(b)))
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("Service health is OK again", func(t *testing.T) {
		// GIVEN
		req := httptest.NewRequest(http.MethodGet, "/api/v1/healthcheck", nil)
		resp := httptest.NewRecorder()

		// WHEN
		handler.ServeHTTP(resp, req)

		// THEN
		assert.Equal(t, http.StatusOK, resp.Code)
	})
}
