package main

type Send struct{
	peerId int
	event interface{}
}

type Commit struct{
	index int
	data [] byte
	error string
}

type Alarm struct{
	t int
}

type LogStore struct{
	index int
	logData LogEntry
}



