package secondary

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"replicated-log/internal/model"
	"replicated-log/internal/storage"
	"time"
)

type HttpHandler struct {
	storage *storage.InMemoryStorage
}

type GetMessagesResponse struct {
	Messages []string `json:"messages"`
}

func (h *HttpHandler) ReplicateMessage(rw http.ResponseWriter, r *http.Request) {
	var message model.Message

	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received message %d with content '%s'\n", message.Id, message.Message)
	isAdded := h.storage.AddMessage(message)
	log.Printf("Added message %d to the storage: %t\n", message.Id, isAdded)

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

	r.HandleFunc("/api/v1/replicate", handler.ReplicateMessage).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/messages", handler.GetMessages)

	return r
}

func NewSecondaryServer() *http.Server {
	handler := &HttpHandler{
		storage: storage.NewInMemoryStorage(),
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
