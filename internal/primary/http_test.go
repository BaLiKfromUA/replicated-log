package primary

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"replicated-log/internal/model"
	"strings"
	"testing"
)

func TestAppendMessage(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "Test"}
	b, _ := json.Marshal(message)

	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var actualMessage model.Message
		err := json.NewDecoder(r.Body).Decode(&actualMessage)
		// THEN
		assert.NoError(t, err)
		assert.Equal(t, message, actualMessage)
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
