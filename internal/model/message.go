package model

type MessageId int32 // just to make future type replacement easy

// Message -- basic struct to represent messages which we want to replicate
type Message struct {
	Id      MessageId `json:"id"` // order of message
	Message string    `json:"message"`
}
