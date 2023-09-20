package secondary

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/suite"
	"io"
	"log"
	"net/http"
	"os"
	"replicated-log/internal/model"
	"strings"
	"testing"

	openapi3routers "github.com/getkin/kin-openapi/routers"
	openapi3legacy "github.com/getkin/kin-openapi/routers/legacy"
)

//go:embed secondary.yaml
var apiSpec []byte
var ctx = context.Background()

type SecondaryApiSuite struct {
	suite.Suite

	server *http.Server

	client        http.Client
	apiSpecRouter openapi3routers.Router
}

func TestSecondaryAPI(t *testing.T) {
	suite.Run(t, &SecondaryApiSuite{})
}

func (s *SecondaryApiSuite) TestReplication() {
	s.Run("Initial message list is empty", func() {
		// when
		resp, err := s.client.Get("http://localhost:8081/api/v1/messages")

		// then
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		s.Require().NoError(err)
		s.Require().Equal("{\"messages\":[]}", string(body))
	})

	s.Run("Replication of a new message", func() {
		// given
		message := model.Message{Id: 0, Message: "Test"}
		b, _ := json.Marshal(message)
		r := io.NopCloser(strings.NewReader(string(b)))

		// when
		resp, err := s.client.Post("http://localhost:8081/api/v1/replicate", "application/json", r)

		// then
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
	})

	s.Run("Update message list contains replicated message", func() {
		// when
		resp, err := s.client.Get("http://localhost:8081/api/v1/messages")

		// then
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		s.Require().NoError(err)
		s.Require().Equal("{\"messages\":[\"Test\"]}", string(body))
	})
}

// setup/teardown
func (s *SecondaryApiSuite) SetupSuite() {
	spec, err := openapi3.NewLoader().LoadFromData(apiSpec)
	s.Require().NoError(err)
	s.Require().NoError(spec.Validate(ctx))
	router, err := openapi3legacy.NewRouter(spec)
	s.Require().NoError(err)
	s.apiSpecRouter = router
	s.client.Transport = s.specValidating(http.DefaultTransport)
}

func (s *SecondaryApiSuite) BeforeTest(_, _ string) {
	s.T().Setenv("SECONDARY_SERVER_PORT", "8081")
	s.server = NewSecondaryServer()
	serviceRunning := make(chan struct{}, 1)
	go func() {
		log.Printf("Start serving on %s", s.server.Addr)
		close(serviceRunning)
		log.Println(s.server.ListenAndServe())
	}()
	<-serviceRunning
}

func (s *SecondaryApiSuite) AfterTest() {
	s.Require().NoError(s.server.Shutdown(ctx))
}

// Helpers
func (s *SecondaryApiSuite) specValidating(transport http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		log.Println("Send HTTP request:")
		reqBody := s.printReq(req)

		// validate request
		route, params, err := s.apiSpecRouter.FindRoute(req)
		s.Require().NoError(err)
		reqDescriptor := &openapi3filter.RequestValidationInput{
			Request:     req,
			PathParams:  params,
			QueryParams: req.URL.Query(),
			Route:       route,
		}
		s.Require().NoError(openapi3filter.ValidateRequest(ctx, reqDescriptor))

		// do request
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		resp, err := transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		log.Println("Got HTTP response:")
		respBody := s.printResp(resp)

		// Validate response against OpenAPI spec
		s.Require().NoError(openapi3filter.ValidateResponse(ctx, &openapi3filter.ResponseValidationInput{
			RequestValidationInput: reqDescriptor,
			Status:                 resp.StatusCode,
			Header:                 resp.Header,
			Body:                   io.NopCloser(bytes.NewReader(respBody)),
		}))

		return resp, nil
	})
}

func (s *SecondaryApiSuite) printReq(req *http.Request) []byte {
	body := s.readAll(req.Body)

	req.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(req.Write(os.Stdout))
	fmt.Println()

	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *SecondaryApiSuite) printResp(resp *http.Response) []byte {
	body := s.readAll(resp.Body)

	resp.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(resp.Write(os.Stdout))
	fmt.Println()

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *SecondaryApiSuite) readAll(in io.Reader) []byte {
	if in == nil {
		return nil
	}
	data, err := io.ReadAll(in)
	s.Require().NoError(err)
	return data
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
