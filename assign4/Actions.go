package main

type Send struct {
	PeerId int
	Event  interface{}
}

type Commit struct {
	Index int
	Data  []byte
	Err   error
}

type Alarm struct {
	t int
}

type LogStore struct {
	Index   int
	LogData LogEntry
}

type StateStore struct {
	State    string
	Term     int
	VotedFor int
}
