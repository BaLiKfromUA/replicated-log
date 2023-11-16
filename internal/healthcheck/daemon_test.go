package healthcheck

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIfMonitoringDaemonDoesHealthCheckAtMomentOfCreation(t *testing.T) {
	// GIVEN
	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer secondary.Close()

	// WHEN
	daemon := NewMonitoringDaemon([]string{secondary.URL})

	// THEN
	require.Equal(t, daemon.GetStatus(secondary.URL), ALIVE)
}

func TestIfBadResponseSetsHealthStatusToDead(t *testing.T) {
	// GIVEN
	calls := 0
	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		if calls == 0 {
			rw.WriteHeader(http.StatusOK)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
		}
		calls += 1
	}))
	defer secondary.Close()
	daemon := NewMonitoringDaemon([]string{secondary.URL})

	// WHEN
	before := daemon.GetStatus(secondary.URL)
	daemon.checkHealth(secondary.URL)
	after := daemon.GetStatus(secondary.URL)

	// THEN
	require.Equal(t, calls, 2)

	require.Equal(t, before, ALIVE)
	require.Equal(t, after, DEAD)
}

func TestNoQuorumReturnsTrueIfAllSecondariesAreDead(t *testing.T) {
	// GIVEN
	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer secondary.Close()

	// WHEN
	daemon := NewMonitoringDaemon([]string{secondary.URL})

	// THEN
	require.True(t, daemon.NoQuorum())
}

func TestNoQuorumReturnsFalseIfAtLeastOneSecondaryIsAlive(t *testing.T) {
	// GIVEN
	liveSecondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer liveSecondary.Close()

	deadSecondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer deadSecondary.Close()

	// WHEN
	daemon := NewMonitoringDaemon([]string{liveSecondary.URL, deadSecondary.URL})

	// THEN
	require.False(t, daemon.NoQuorum())
}
