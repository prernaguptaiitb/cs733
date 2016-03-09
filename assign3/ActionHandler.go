package main
import "time"

func (rn *RaftNode) AlarmHandler (ac Alarm){
	time.Sleep(time.Duration(ac.t) * time.Millisecond)
    (rn.TimeoutCh) <- TimeoutEvent{}
}

func (rn *RaftNode) SendHandler (ac interface{}){

}

func (rn *RaftNode) CommitHandler (ac interface{}){

}

func (rn *RaftNode) LogStoreHandler (ac interface{}){

}

func (rn *RaftNode) StateStoreHandler (ac interface{}){

}
