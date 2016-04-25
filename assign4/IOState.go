package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
)

func OpenFile(filename string) *os.File {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

func CloseFile(f *os.File) {
	f.Close()
}

func WriteState(f *os.File, ac StateStore) {
	//   f:=OpenFile(filename)
	b := bufio.NewWriter(f)
	defer func() {
		if err := b.Flush(); err != nil {
			log.Fatal(err)
		}
	}()
	state := ac.State
	currentTerm := strconv.Itoa(ac.Term)
	votedFor := strconv.Itoa(ac.VotedFor)
	_, err := fmt.Fprintf(b, "%s %s %s\n", state, currentTerm, votedFor)
	if err != nil {
		log.Fatal(err)
	}

}

func ReadState(f *os.File) SMState {
	var smstate SMState
	b := bufio.NewReader(f)
	var state, currentTerm, votedFor string
	_, err := fmt.Fscanf(b, "%s %s %s\n", &state, &currentTerm, &votedFor)
	if err != nil {
		log.Fatal(err)
	}
	smstate.State = state
	smstate.CurrentTerm, err = strconv.Atoi(currentTerm)
	if err != nil {
		log.Fatal(err)
	}
	smstate.VotedFor, err = strconv.Atoi(votedFor)
	if err != nil {
		log.Fatal(err)
	}
	return smstate
}
