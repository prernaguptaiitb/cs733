package main

type VoteRequestEvent struct{
	term int
	candidateId int
	lastLogIndex int
	lastLogTerm int
}

func (sm *StateMachine) VoteRequest (msg VoteRequestEvent) ([] interface{}){
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
func (sm *StateMachine) VoteRequestLeaderorCandidate (msg VoteRequestEvent) ([] interface{}){
	var action []interface{}
	if sm.currentTerm < msg.term {
		// Update the term and change to follower state
		sm.currentTerm=msg.term;
		sm.state="FOLLOWER"
		//if candidate log is at least as up-to-date as this leaders log, then grant vote otherwise not
		if (sm.log[sm.logCurrentIndex].term < msg.lastLogTerm) || ((sm.log[sm.logCurrentIndex].term == msg.lastLogTerm) && (sm.logCurrentIndex <= msg.lastLogIndex)){
			action = append(action, Send{peerId : msg.candidateId, event : VoteResponseEvent{term : sm.currentTerm, isVoteGranted : true}})
			sm.votedFor = msg.candidateId
		}else{
			// do not grant vote
			action = append(action, Send{peerId : msg.candidateId, event : VoteResponseEvent{term : sm.currentTerm, isVoteGranted : false}})
		}
		action = append(action, Alarm{t : 200})	
	}else{
		// reject vote
		action = append(action, Send{peerId : msg.candidateId, event : VoteResponseEvent{term : sm.currentTerm, isVoteGranted : false}})
	}
	return action
}

func (sm *StateMachine) VoteRequestFollower (msg VoteRequestEvent) ([] interface{}){
	var action []interface{}
	if sm.currentTerm < msg.term && (sm.votedFor == 0 /*assuming no server has id=0 */|| sm.votedFor == msg.candidateId) {  
		sm.currentTerm = msg.term 
	//if candidate log is at least as up-to-date as this leaders log, then grant vote otherwise not
		if (sm.log[sm.logCurrentIndex].term < msg.lastLogTerm) || ((sm.log[sm.logCurrentIndex].term == msg.lastLogTerm) && (sm.logCurrentIndex <= msg.lastLogIndex)){
			action = append(action, Send{peerId : msg.candidateId, event : VoteResponseEvent{term : sm.currentTerm, isVoteGranted : true}})
			sm.votedFor = msg.candidateId
			action = append(action, Alarm{t : 200})	
		}else{
			// do not grant vote
			action = append(action, Send{peerId : msg.candidateId, event : VoteResponseEvent{term : sm.currentTerm, isVoteGranted : false}})
		}
	}else{
		// reject vote
		action = append(action, Send{peerId : msg.candidateId, event : VoteResponseEvent{term : sm.currentTerm, isVoteGranted : false}})

	}
	return action
}
