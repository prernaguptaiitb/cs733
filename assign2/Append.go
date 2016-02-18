package main

type AppendEvent struct{
	data [] byte
}

func (sm *StateMachine) Append (msg AppendEvent) ([] interface{}){
	var action []interface{}
	switch sm.state {
				case "LEADER":
					action = sm.AppendLeader(msg)			
				case "FOLLOWER":
					action = sm.AppendFollower(msg)
				case "CANDIDATE":
					action = sm.AppendCandidate(msg)
			}
		return action
}


func (sm *StateMachine) AppendLeader (msg AppendEvent) ([] interface{}){
	var action [] interface{}
	// append data to leaders local log
	/*sm.logCurrentIndex++
	temp := LogEntry{term : sm.currentTerm, cmd : data }
	action = append(action,LogStore { index : sm.logCurrentIndex , logData : temp})
	//send AppendEntriesRequest to all peers
	for i int =0; i<len(sm.myconfig.peer); i++ { 
		action = append(action,Send{peerId : sm.myconfig.peer[i], event : AppendEntriesRequestEvent{ term : sm.currentTerm, leaderId : config.myId, prevLogIndex : sm.logCurrentIndex-1, prevLogTerm : sm.log[sm.logCurrentIndex-1].term, data : sm.log[sm.logCurrentIndex], leaderCommitIndex : sm.logCommitIndex}})
	}



	
	Check if there exists an N such that N > sm.logCommitIndex, a majority of sm.matchIndex[i] ≥ N, and sm.log[N].term == sm.currentTerm:
set sm.logCommitIndex = N
	Commit(sm.logCommitIndex, data, NIL)
else 
	Commit(sm.logCommitIndex, data, error)
*/
	return action
}


func (sm *StateMachine) AppendFollower (msg AppendEvent) ([] interface{}){
	var action []interface{}
	return action
}


func (sm *StateMachine) AppendCandidate (msg AppendEvent) ([] interface{}){
	var action []interface{}
	return action
}


