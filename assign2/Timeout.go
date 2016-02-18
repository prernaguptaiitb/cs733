package main

type Timeout struct{

}

func (sm *StateMachine) Timeout (msg AppendEntriesRequestEvent) ([] interface{}){
	var action []interface{}
	switch sm.state {
				case "LEADER":
					action = sm.AppendEntriesRequestLeaderorCandidate(msg)			
				case "FOLLOWER":
					action = sm.AppendEntriesRequestFollower(msg)
				case "CANDIDATE":
					action = sm.AppendEntriesRequestLeaderorCandidate(msg)
			}
		return action
}

