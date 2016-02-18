package main
import (
	"fmt"
)

type Config struct{
	myId int	//persistent
	peer [] int		//persistent
}

type StateMachine struct{
	myconfig Config
	state string
	currentTerm int //persistent
	votedFor int	//persistent
	log [] LogEntry //persistent
	logCurrentIndex int
	logCommitIndex int 
	nextIndex	[] int
	matchIndex	[] int
}

type LogEntry struct{
	term int
	cmd [] byte
}

func (sm *StateMachine) checkState (ev interface{}){

}

func main() {}


func (sm *StateMachine) ProcessEvent (ev interface{}) {

	switch ev.(type) {	

		case AppendEvent:
			cmd := ev.(AppendEvent)
			fmt.Printf("%v\n",cmd)

		case AppendEntriesRequestEvent:
			cmd := ev.(AppendEntriesRequestEvent)
			sm.AppendEntriesRequest(cmd)
			fmt.Printf("%v\n", cmd)
		
		case AppendEntriesResponseEvent:
			cmd := ev.(AppendEntriesResponseEvent)
			fmt.Printf("%v\n",cmd)
		
		case VoteRequestEvent:
			cmd := ev.(VoteRequestEvent)
			fmt.Printf("%v\n", cmd)
		
		case VoteResponseEvent:
			cmd := ev.(VoteResponseEvent)
			fmt.Printf("%v\n",cmd)
		
		case Timeout:
			cmd := ev.(Timeout)
			fmt.Printf("%v\n",cmd)
		
		// other cases
		default: println ("Unrecognized")
	}
}