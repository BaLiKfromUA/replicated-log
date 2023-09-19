package replication

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"replicated-log/internal/model"
	"testing"
)

func TestReplicateMessageWithOneSecondary(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "first one"}

	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var actualMessage model.Message
		err := json.NewDecoder(r.Body).Decode(&actualMessage)
		// THEN
		require.NoError(t, err)
		require.Equal(t, message, actualMessage)
		rw.WriteHeader(http.StatusOK)
	}))
	defer secondary.Close()

	t.Setenv("SECONDARY_URLS", secondary.URL)

	// WHEN
	NewExecutor().ReplicateMessage(message)
}

func TestReplicateMessageWithTwoSecondaries(t *testing.T) {
	// GIVEN
	message := model.Message{Id: 0, Message: "first one"}

	handler := func(rw http.ResponseWriter, r *http.Request) {
		var actualMessage model.Message
		err := json.NewDecoder(r.Body).Decode(&actualMessage)
		// THEN
		require.NoError(t, err)
		require.Equal(t, message, actualMessage)
		rw.WriteHeader(http.StatusOK)
	}

	secondaryA := httptest.NewServer(http.HandlerFunc(handler))
	defer secondaryA.Close()
	secondaryB := httptest.NewServer(http.HandlerFunc(handler))
	defer secondaryB.Close()

	t.Setenv("SECONDARY_URLS", secondaryA.URL+","+secondaryB.URL)

	// WHEN
	NewExecutor().ReplicateMessage(message)
}