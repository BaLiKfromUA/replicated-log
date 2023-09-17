package storage

import (
	"github.com/stretchr/testify/assert"
	"replicated-log/internal/model"
	"testing"
)

func TestAddMessage(t *testing.T) {
	storage := NewInMemoryStorage()
	message := model.Message{Id: 0, Message: "test"}

	t.Run("New message is added successfully", func(t *testing.T) {
		isAdded := storage.AddMessage(message)
		assert.True(t, isAdded)
	})

	t.Run("Item message the same ID cannot be added again", func(t *testing.T) {
		isAdded := storage.AddMessage(message)
		assert.False(t, isAdded)
	})
}

func TestGetMessages(t *testing.T) {
	storage := NewInMemoryStorage()

	t.Run("Messages in order are successfully returned from storage", func(t *testing.T) {
		storage.AddMessage(model.Message{Id: 0, Message: "first one"})
		storage.AddMessage(model.Message{Id: 1, Message: "second"})
		storage.AddMessage(model.Message{Id: 2, Message: "third"})

		messages := storage.GetMessages()
		assert.Equal(t, []string{"first one", "second", "third"}, messages)
	})

	t.Run("Message out of order is not returned from storage", func(t *testing.T) {
		storage.AddMessage(model.Message{Id: 4, Message: "Five is out of order"})

		messages := storage.GetMessages()
		assert.Equal(t, []string{"first one", "second", "third"}, messages, "Message with id 4 is not expected in this list")
	})

	t.Run("Missing message restore the total order", func(t *testing.T) {
		storage.AddMessage(model.Message{Id: 3, Message: "Four"})

		messages := storage.GetMessages()
		assert.Equal(t, []string{"first one", "second", "third", "Four", "Five is out of order"}, messages)
	})
}
