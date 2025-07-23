package models

type User struct {
	Bio string
	Nickname string
	Username string
}

type Message struct {
	ID int64
	Sender string
	Text string
}

type Chat struct {
	Name string
	Participants string
	Messages []Message
}