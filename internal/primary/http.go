package primary

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"replicated-log/internal/replication"
	"replicated-log/internal/storage"
	"time"
)

type HttpHandler struct {
	storage  *storage.InMemoryStorage
	executor *replication.Executor
}

type AppendMessageRequest struct {
	Message string `json:"message"`
	W       int    `json:"w"`
}

type GetMessagesResponse struct {
	Messages []string `json:"messages"`
}

func (h *HttpHandler) AppendMessage(rw http.ResponseWriter, r *http.Request) {
	var payload AppendMessageRequest

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	message := h.storage.AddRawMessage(payload.Message)
	h.executor.ReplicateMessage(message, payload.W-1)
	rw.WriteHeader(http.StatusOK)
}

func (h *HttpHandler) GetMessages(rw http.ResponseWriter, _ *http.Request) {
	messages := h.storage.GetMessages()
	if messages == nil {
		messages = []string{}
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rawResponse, _ := json.Marshal(GetMessagesResponse{Messages: messages})
	_, _ = rw.Write(rawResponse)
}

func createRouter(handler *HttpHandler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/append", handler.AppendMessage).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/messages", handler.GetMessages)

	return r
}

func NewPrimaryServer() *http.Server {
	handler := &HttpHandler{
		storage:  storage.NewInMemoryStorage(),
		executor: replication.NewExecutor(),
	}

	port, ok := os.LookupEnv("PRIMARY_SERVER_PORT")
	if !ok {
		port = "8000"
	}

	srv := &http.Server{
		Handler:      createRouter(handler),
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return srv
}
