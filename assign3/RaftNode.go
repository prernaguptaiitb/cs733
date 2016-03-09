package main


type RaftNode struct { 
	// implements Node interface
	sm StateMachine 
	EventCh chan interface{} 
	TimeoutCh chan TimeoutEvent //for timeout events
	CommitCh chan CommitInfo
}

// data goes in via Append, comes out as CommitInfo from the node's CommitChannel
// Index is valid only if err == nil
type CommitInfo struct {
	Data []byte
	Index int64 // or int .. whatever you have in your code
	Err error // Err can be errred
}

func (rn *RaftNode) Append(data []byte) {
	rn.EventCh <- AppendEvent{data: data}
}

func (rn *RaftNode) CommitChannel() <-chan CommitInfo{
	return CommitCh
}


func (rn *RaftNode) processEvents() {
	var actions []interface{}
	for {
		var ev interface{}
		select {
			case ev = <- rn.EventCh :
			case <- rn.TimeoutCh :
		}
	actions = rn.sm.ProcessEvent(ev)	
	rn.doActions(actions)
	}	
}

func (rn *RaftNode) doActions(actions []interface{}){
	var ac interface{}
	for i:= 0; i < len(actions); i++ {
			ac = actions[i]
			switch ac.(type) {
				case Alarm:
					res := ac.(Alarm)
					go rn.AlarmHandler(res)
				case Send:
					res := ac.(Send)
					rn.SendHandler(res)
				case Commit:
					res := ac.(Commit)
					rn.CommitHandler(res)
				case LogStore:
					res := ac.(LogStore)
					rn.LogStoreHandler(res)
				case StateStore:
					res := ac.(StateStore)
					rn.StateStoreHandler(res)
			}
	 }
}




/*
// This is an example structure for Config .. change it to your convenience.
type struct Config {
	cluster []NetConfig // Information about all servers, including this.
	Id int // this node's id. One of the cluster's entries should match.
	LogDir string // Log file directory for this node
	ElectionTimeout int
	HeartbeatTimeout int
}

type struct NetConfig {
	Id int
	Host string
	Port int
}

*/



