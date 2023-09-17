package storage

import (
	"log"
	"replicated-log/internal/model"
	"sync"
)

type InMemoryStorage struct {
	mu   *sync.Mutex
	data map[model.MessageId]string
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		mu:   &sync.Mutex{},
		data: make(map[model.MessageId]string),
	}
}

func (s *InMemoryStorage) AddMessage(message model.Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[message.Id]; ok {
		// All messages should be present exactly once in the secondary log - deduplication
		log.Printf("Message %d already exists", message.Id)
		return false
	}

	s.data[message.Id] = message.Message

	return true
}

func (s *InMemoryStorage) GetMessages() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastId := model.MessageId(len(s.data) - 1)

	var result []string

	for id := model.MessageId(0); id <= lastId; id++ {
		value, ok := s.data[id]
		if !ok {
			// If secondary has received messages [msg1, msg2, msg4], it shouldn’t display the message ‘msg4’ until the ‘msg3’ will be received
			log.Printf("Message %d is missing, stop getting next messages", id)
			break
		}
		result = append(result, value)
	}

	return result
}
