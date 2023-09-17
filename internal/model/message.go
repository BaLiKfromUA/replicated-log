package model

type MessageId uint32 // just to make future type replacement easy

// Message -- basic struct to represent messages which we want to replicate
type Message struct {
	Id      MessageId `json:"order"`
	Message string    `json:"message"`
}
