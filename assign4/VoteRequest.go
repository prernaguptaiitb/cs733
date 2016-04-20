package main

//import "fmt"

type VoteRequestEvent struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

func (sm *StateMachine) VoteRequest(msg VoteRequestEvent) []interface{} {
	var action []interface{}
	switch sm.state {
	case "LEADER":
		action = sm.VoteRequestLeaderorCandidate(msg)
	case "FOLLOWER":
		action = sm.VoteRequestFollower(msg)
	case "CANDIDATE":
		action = sm.VoteRequestLeaderorCandidate(msg)
	}
	return action
}
func (sm *StateMachine) VoteRequestLeaderorCandidate(msg VoteRequestEvent) []interface{} {
	var action []interface{}
	if sm.currentTerm < msg.Term {
		// Update the Term and change to follower state
		sm.currentTerm = msg.Term
		if sm.state == "LEADER"{
			actionsPending := sm.PendingRequest()
			action = append(action, actionsPending...)
		}
		sm.state = "FOLLOWER"
		sm.votedFor = 0
		
		//if candidate log is at least as up-to-date as this leaders log, then grant vote otherwise not
		var LastlogTerm int
		if sm.logCurrentIndex == -1 {
			LastlogTerm = 0
		} else {
			LastlogTerm = sm.log[sm.logCurrentIndex].Term
		}
		action = append(action, Alarm{t: Random(sm.electionTO)})
		if (LastlogTerm < msg.LastLogTerm) || ((LastlogTerm == msg.LastLogTerm) && (sm.logCurrentIndex <= msg.LastLogIndex)) {
			//grant vote
			sm.votedFor = msg.CandidateId
			
			action = append(action, Send{PeerId: msg.CandidateId, Event: VoteResponseEvent{Term: sm.currentTerm, IsVoteGranted: true}})

		} else {
			// do not grant vote
			action = append(action, Send{PeerId: msg.CandidateId, Event: VoteResponseEvent{Term: sm.currentTerm, IsVoteGranted: false}})
		}
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		
	} else {
		// reject vote
		action = append(action, Send{PeerId: msg.CandidateId, Event: VoteResponseEvent{Term: sm.currentTerm, IsVoteGranted: false}})
	}
	return action
}

func (sm *StateMachine) VoteRequestFollower(msg VoteRequestEvent) []interface{} {

	var action []interface{}
	flag := false
	if (sm.currentTerm == msg.Term && (sm.votedFor == 0 /*assuming no server has id=0 */ || sm.votedFor == msg.CandidateId)) || (sm.currentTerm < msg.Term) {
		if sm.currentTerm < msg.Term {
			sm.currentTerm = msg.Term
			sm.votedFor = 0
			flag = true
		}
		//if candidate log is at least as up-to-date as this leaders log, then grant vote otherwise not
		var LastlogTerm int
		if sm.logCurrentIndex == -1 {
			LastlogTerm = 0
		} else {
			LastlogTerm = sm.log[sm.logCurrentIndex].Term
		}
		if (LastlogTerm < msg.LastLogTerm) || ((LastlogTerm == msg.LastLogTerm) && (sm.logCurrentIndex <= msg.LastLogIndex)) {
			// grant vote
			sm.votedFor = msg.CandidateId
			if sm.currentTerm < msg.Term {
				flag = true
			}
			//		fmt.Printf("ID :%v Alarm increased \n ", sm.myconfig.myId)
			action = append(action, Alarm{t: Random(sm.electionTO)})
			action = append(action, Send{PeerId: msg.CandidateId, Event: VoteResponseEvent{Term: sm.currentTerm, IsVoteGranted: true}})
		} else {
			// do not grant vote
			action = append(action, Send{PeerId: msg.CandidateId, Event: VoteResponseEvent{Term: sm.currentTerm, IsVoteGranted: false}})
		}
		if flag {
			action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		}
	} else {
		// reject vote
		action = append(action, Send{PeerId: msg.CandidateId, Event: VoteResponseEvent{Term: sm.currentTerm, IsVoteGranted: false}})

	}
	return action
}
