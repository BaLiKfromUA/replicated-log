package secondary

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"replicated-log/internal/model"
	"replicated-log/internal/storage"
	"replicated-log/internal/util"
	"time"
)

type HttpHandler struct {
	storage  *storage.InMemoryStorage
	emulator *util.InconsistencyEmulator
}

type GetMessagesResponse struct {
	Messages []string `json:"messages"`
}

type SwitchReplicationModeRequest struct {
	ShouldWait bool `json:"enable"`
}

func (h *HttpHandler) ReplicateMessage(rw http.ResponseWriter, r *http.Request) {
	var message model.Message

	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received message %d with content '%s'\n", message.Id, message.Message)
	h.emulator.BlockRequestIfNeeded()
	isAdded := h.storage.AddMessage(message)
	log.Printf("Added message %d to the storage: %t\n", message.Id, isAdded)

	rw.WriteHeader(http.StatusOK)
}

func (h *HttpHandler) GetMessages(rw http.ResponseWriter, _ *http.Request) {
	messages := h.storage.GetMessages()

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rawResponse, _ := json.Marshal(GetMessagesResponse{Messages: messages})
	log.Printf("Get messages: %v", messages)
	_, _ = rw.Write(rawResponse)
}

func (h *HttpHandler) CleanStorage(rw http.ResponseWriter, _ *http.Request) {
	h.storage.Clear()
	rw.WriteHeader(http.StatusOK)
}

func (h *HttpHandler) SwitchReplicationMode(rw http.ResponseWriter, r *http.Request) {
	var body SwitchReplicationModeRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	h.emulator.ChangeMode(body.ShouldWait)
	rw.WriteHeader(http.StatusOK)
}

func (h *HttpHandler) HealthCheck(rw http.ResponseWriter, _ *http.Request) {
	if h.emulator.IsShouldWait() {
		rw.WriteHeader(http.StatusNotAcceptable)
	} else {
		rw.WriteHeader(http.StatusOK)
	}
}

func createRouter(handler *HttpHandler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/api/v1/internal/replicate", handler.ReplicateMessage).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/messages", handler.GetMessages).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/healthcheck", handler.HealthCheck).Methods(http.MethodGet)

	r.HandleFunc("/api/test/clean", handler.CleanStorage).Methods(http.MethodPost)
	r.HandleFunc("/api/test/replication_block", handler.SwitchReplicationMode).Methods(http.MethodPost)

	return r
}

func NewSecondaryServer() *http.Server {
	handler := &HttpHandler{
		storage:  storage.NewInMemoryStorage(),
		emulator: util.NewInconsistencyEmulator(),
	}

	port, ok := os.LookupEnv("SECONDARY_SERVER_PORT")
	if !ok {
		port = "8080"
	}

	srv := &http.Server{
		Handler:      createRouter(handler),
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return srv
}
