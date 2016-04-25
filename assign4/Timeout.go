package main

//import "fmt"

type TimeoutEvent struct {
}

func (sm *StateMachine) Timeout(msg TimeoutEvent) []interface{} {
	var action []interface{}
	switch sm.state {
	case "LEADER":
		action = sm.TimeoutLeader(msg)
	case "FOLLOWER":
		action = sm.TimeoutFollower(msg)
	case "CANDIDATE":
		action = sm.TimeoutCandidate(msg)
	}
	return action
}
func (sm *StateMachine) TimeoutLeader(msg TimeoutEvent) []interface{} {
	//	fmt.Printf("Heartbeat Timeout ID: %v, Term : %v \n" , sm.myconfig.myId, sm.currentTerm)
	var action []interface{}
	action = append(action, Alarm{t: sm.heartbeatTO})
	//heartbeat timeout
	for i := 0; i < len(sm.myconfig.peer); i++ {
		if sm.nextIndex[i] == 0 {
			action = append(action, Send{PeerId: sm.myconfig.peer[i], Event: AppendEntriesRequestEvent{Term: sm.currentTerm, LeaderId: sm.myconfig.myId, PrevLogIndex: sm.nextIndex[i] - 1, PrevLogTerm: 0, Data: sm.log[sm.nextIndex[i]:], LeaderCommitIndex: sm.logCommitIndex}})
		} else {
			action = append(action, Send{PeerId: sm.myconfig.peer[i], Event: AppendEntriesRequestEvent{Term: sm.currentTerm, LeaderId: sm.myconfig.myId, PrevLogIndex: sm.nextIndex[i] - 1, PrevLogTerm: sm.log[sm.nextIndex[i]-1].Term, Data: sm.log[sm.nextIndex[i]:], LeaderCommitIndex: sm.logCommitIndex}})
		}
	}
	return action
}
func (sm *StateMachine) TimeoutFollower(msg TimeoutEvent) []interface{} {
	var action []interface{}
	//election timeout
	sm.currentTerm += 1
	//	fmt.Printf("Election Timeout ID : %v, Term : %v \n" , sm.myconfig.myId, sm.currentTerm)
	sm.state = "CANDIDATE"
	sm.votedFor = sm.myconfig.myId
	action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
	sm.yesVotesNum = 1
	sm.noVotesNum = 0
	//Reset election timer
	action = append(action, Alarm{t: Random(sm.electionTO)})
	for i := 0; i < len(sm.myconfig.peer); i++ {
		if sm.logCurrentIndex == -1 {
			action = append(action, Send{PeerId: sm.myconfig.peer[i], Event: VoteRequestEvent{Term: sm.currentTerm, CandidateId: sm.myconfig.myId, LastLogIndex: sm.logCurrentIndex, LastLogTerm: 0}})
		} else {
			action = append(action, Send{PeerId: sm.myconfig.peer[i], Event: VoteRequestEvent{Term: sm.currentTerm, CandidateId: sm.myconfig.myId, LastLogIndex: sm.logCurrentIndex, LastLogTerm: sm.log[sm.logCurrentIndex].Term}})
		}

	}
	return action
}
func (sm *StateMachine) TimeoutCandidate(msg TimeoutEvent) []interface{} {
	var action []interface{}
	//election timeout
	sm.currentTerm += 1
	sm.votedFor = sm.myconfig.myId
	action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
	sm.yesVotesNum = 1
	sm.noVotesNum = 0
	//Reset election timer
	action = append(action, Alarm{t: Random(sm.electionTO)})
	var lastlogTerm int
	for i := 0; i < len(sm.myconfig.peer); i++ {
		if sm.logCurrentIndex == -1 {
			lastlogTerm = 0
		} else {
			lastlogTerm = sm.log[sm.logCurrentIndex].Term
		}
		action = append(action, Send{PeerId: sm.myconfig.peer[i], Event: VoteRequestEvent{Term: sm.currentTerm, CandidateId: sm.myconfig.myId, LastLogIndex: sm.logCurrentIndex, LastLogTerm: lastlogTerm}})
	}
	return action
}
