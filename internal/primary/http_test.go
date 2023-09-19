package primary

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	openapi3routers "github.com/getkin/kin-openapi/routers"
	openapi3legacy "github.com/getkin/kin-openapi/routers/legacy"
	"github.com/stretchr/testify/suite"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"replicated-log/internal/model"
	"strings"
	"testing"
)

//go:embed primary.yaml
var apiSpec []byte
var ctx = context.Background()

type PrimaryApiSuite struct {
	suite.Suite

	primary *http.Server

	client        http.Client
	apiSpecRouter openapi3routers.Router
}

func TestPrimaryAPI(t *testing.T) {
	suite.Run(t, &PrimaryApiSuite{})
}

func (s *PrimaryApiSuite) TestAppendMessage() {
	// GIVEN
	message := model.Message{Id: 0, Message: "Test"}
	b, _ := json.Marshal(message)
	reqBody := io.NopCloser(strings.NewReader(string(b)))

	secondary := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var actualMessage model.Message
		err := json.NewDecoder(r.Body).Decode(&actualMessage)
		// THEN
		s.Require().NoError(err)
		s.Require().Equal(message, actualMessage)
		rw.WriteHeader(http.StatusOK)
	}))
	defer secondary.Close()

	s.T().Setenv("SECONDARY_URLS", secondary.URL)

	s.primary = NewPrimaryServer()
	go func() {
		log.Printf("Start serving on %s", s.primary.Addr)
		log.Println(s.primary.ListenAndServe())
	}()

	s.Run("Initial message list is empty", func() {
		// WHEN
		resp, err := s.client.Get("http://localhost:8000/api/v1/messages")

		// THEN
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		s.Require().NoError(err)
		s.Require().Equal("{\"messages\":[]}", string(body))
	})

	s.Run("Append a new message", func() {
		// WHEN
		resp, err := s.client.Post("http://localhost:8000/api/v1/append", "application/json", reqBody)

		// THEN
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
	})

	s.Run("Update message list contains appended message", func() {
		// WHEN
		resp, err := s.client.Get("http://localhost:8000/api/v1/messages")

		// THEN
		s.Require().NoError(err)
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		s.Require().NoError(err)
		s.Require().Equal("{\"messages\":[\"Test\"]}", string(body))
	})
}

// setup/teardown
func (s *PrimaryApiSuite) AfterTest() {
	s.Require().NoError(s.primary.Shutdown(ctx))
}

func (s *PrimaryApiSuite) SetupSuite() {
	spec, err := openapi3.NewLoader().LoadFromData(apiSpec)
	s.Require().NoError(err)
	s.Require().NoError(spec.Validate(ctx))
	router, err := openapi3legacy.NewRouter(spec)
	s.Require().NoError(err)
	s.apiSpecRouter = router
	s.client.Transport = s.specValidating(http.DefaultTransport)
}

// Helpers
func (s *PrimaryApiSuite) specValidating(transport http.RoundTripper) http.RoundTripper {
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

func (s *PrimaryApiSuite) printReq(req *http.Request) []byte {
	body := s.readAll(req.Body)

	req.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(req.Write(os.Stdout))
	fmt.Println()

	req.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *PrimaryApiSuite) printResp(resp *http.Response) []byte {
	body := s.readAll(resp.Body)

	resp.Body = io.NopCloser(bytes.NewReader(body))
	s.Require().NoError(resp.Write(os.Stdout))
	fmt.Println()

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func (s *PrimaryApiSuite) readAll(in io.Reader) []byte {
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
