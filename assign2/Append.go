package main
import(
	"errors"
)
type AppendEvent struct{
	data [] byte
}

func (sm *StateMachine) Append (msg AppendEvent) ([] interface{}){
	var action []interface{}
	switch sm.state {
				case "LEADER":
					action = sm.AppendLeader(msg)			
				case "FOLLOWER":
					action = sm.AppendFollowerorCandidate(msg)
				case "CANDIDATE":
					action = sm.AppendFollowerorCandidate(msg)
			}
		return action
}


func (sm *StateMachine) AppendLeader (msg AppendEvent) ([] interface{}){
	var action [] interface{}
	// append data to leaders local log
	sm.logCurrentIndex++
	temp := LogEntry{sm.currentTerm, msg.data }
	sm.log = append(sm.log, temp)
	action = append(action,LogStore { sm.logCurrentIndex , temp})
	//send AppendEntriesRequest to all peers
	for i:=0; i<len(sm.myconfig.peer); i++ { 
		action = append(action, Send{sm.myconfig.peer[i], AppendEntriesRequestEvent{ sm.currentTerm, sm.myconfig.myId, sm.nextIndex[sm.myconfig.peer[i]]-1, sm.log[sm.nextIndex[sm.myconfig.peer[i]]-1].term, sm.log[sm.nextIndex[sm.myconfig.peer[i]]:], sm.logCommitIndex}})
	}

	return action
}


func (sm *StateMachine) AppendFollowerorCandidate (msg AppendEvent) ([] interface{}){
	var action []interface{}
	action = append(action, Commit{ -1, msg.data, errors.New("Error in committing. Not a leader") })
	return action
}





