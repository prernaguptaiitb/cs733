package main

type Config struct{
	myId int	//persistent
	peer [] int		//persistent
}

type StateMachine struct{
	myconfig Config
	state string	//persistent
	currentTerm int //persistent
	votedFor int	//persistent
	log [] LogEntry //persistent
	logCurrentIndex int
	logCommitIndex int 
	nextIndex	[] int
	matchIndex	[] int
	yesVotesNum int
	noVotesNum int
}

type LogEntry struct{
	term int
	cmd [] byte
}

func (sm *StateMachine) checkState (ev interface{}){

}

func main() {}


func (sm *StateMachine) ProcessEvent (ev interface{}) ([]interface{}){
	var action []interface{}
	switch ev.(type) {	

		case AppendEvent:
			cmd := ev.(AppendEvent)
			action =sm.Append(cmd)
		
		case AppendEntriesRequestEvent:
			cmd := ev.(AppendEntriesRequestEvent)
			action =sm.AppendEntriesRequest(cmd)
		
		case AppendEntriesResponseEvent:
			cmd := ev.(AppendEntriesResponseEvent)
			action =sm.AppendEntriesResponse(cmd)
			
		case VoteRequestEvent:
			cmd := ev.(VoteRequestEvent)
			action =sm.VoteRequest(cmd)
			
		case VoteResponseEvent:
			cmd := ev.(VoteResponseEvent)
			action =sm.VoteResponse(cmd)
			
		case TimeoutEvent:
			cmd := ev.(TimeoutEvent)
			action =sm.Timeout(cmd)
			
		// other cases
		default: println ("Unrecognized")
	}
	return action
}