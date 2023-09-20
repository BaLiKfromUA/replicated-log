package internal

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/suite"
	"io"
	"log"
	"net/http"
	"replicated-log/internal/primary"
	"replicated-log/internal/secondary"
	"strings"
	"testing"
)

var ctx = context.Background()

var primaryURL = "http://localhost:8000"
var secondaryURL = "http://localhost:8080"

type PrimarySecondaryIntegrationTestSuite struct {
	suite.Suite

	primary   *http.Server
	secondary *http.Server

	client http.Client
}

func TestPrimarySecondaryIntegration(t *testing.T) {
	suite.Run(t, &PrimarySecondaryIntegrationTestSuite{})
}

func (s *PrimarySecondaryIntegrationTestSuite) TestBasicReplication() {
	s.Run("Append a new message", func() {
		// GIVEN
		req := primary.AppendMessageRequest{Message: "Test"}
		b, _ := json.Marshal(req)
		reqBody := io.NopCloser(strings.NewReader(string(b)))
		// WHEN
		resp, err := s.client.Post(primaryURL+"/api/v1/append", "application/json", reqBody)
		// THEN
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
	})

	// GIVEN
	expectedResp := "{\"messages\":[\"Test\"]}"
	s.Run("Primary contains appended message", func() {
		// WHEN
		resp, err := s.client.Get(primaryURL + "/api/v1/messages")
		// THEN
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		s.Require().NoError(err)
		s.Require().Equal(expectedResp, string(body))
	})

	s.Run("Secondary contains appended message", func() {
		// WHEN
		resp, err := s.client.Get(secondaryURL + "/api/v1/messages")
		// THEN
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		s.Require().NoError(err)
		s.Require().Equal(expectedResp, string(body))
	})
}

// setUp/tearDown
func (s *PrimarySecondaryIntegrationTestSuite) BeforeTest(_, _ string) {
	s.T().Setenv("PRIMARY_SERVER_PORT", "8000")
	s.T().Setenv("SECONDARY_SERVER_PORT", "8080")

	ready := make(chan struct{}, 2)
	defer close(ready)

	s.secondary = secondary.NewSecondaryServer()
	go func() {
		log.Printf("Start serving SECONDARY on %s", s.secondary.Addr)
		ready <- struct{}{}
		log.Println(s.secondary.ListenAndServe())
	}()

	s.T().Setenv("SECONDARY_URLS", secondaryURL)
	s.primary = primary.NewPrimaryServer()
	go func() {
		log.Printf("Start serving PRIMARY on %s", s.primary.Addr)
		ready <- struct{}{}
		log.Println(s.primary.ListenAndServe())
	}()

	// waiting for all services to start in order to ensure that test is not flaky
	for i := 0; i < 2; i++ {
		<-ready
	}
}

func (s *PrimarySecondaryIntegrationTestSuite) AfterTest() {
	s.Require().NoError(s.primary.Shutdown(ctx))
	s.Require().NoError(s.secondary.Shutdown(ctx))
}
