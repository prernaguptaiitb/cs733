package main

import (
	"bytes"
	"bufio"
	"fmt"
	"errors"
	"github.com/prernaguptaiitb/cs733/assign4/fs"
	"encoding/gob"
	"net"
	"os"
	"strconv"
)

var crlf = []byte{'\r', '\n'}
var MAX_CLIENTS int64= 100000000000 // our server can support 100,000,000,000 clients. After this client id will start repeating
//var ClientChanMap map[int]chan CommitInfo

type FSConfig struct {
	Id   int
	Address string
//	Port int
}

type Server struct{
	fsconf []FSConfig
	ClientChanMap map[int64]chan error
	rn RaftNode 
}

type MsgEntry struct {
	Data fs.Msg
}
/*
type ClientEntry struct{
	Msg fs.Msg
	Err error
}*/

func encode(data fs.Msg) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	me := MsgEntry{Data: data}
	err := enc.Encode(me)
	return buf.Bytes(), err
}

func decode(dbytes []byte) (fs.Msg, error) {
	buf := bytes.NewBuffer(dbytes)
	enc := gob.NewDecoder(buf)
	var me MsgEntry
	err := enc.Decode(&me)
	return me.Data, err
}


func check(obj interface{}) {
	if obj != nil {
		fmt.Println(obj)
		os.Exit(1)
	}
}
/*
func assert(val bool) {
	if !val {
		panic("Assertion Failed")
	}
}
*/
func (server *Server) getAddress(id int) string{
//	fmt.Printf("id %v\n", id)
//	fmt.Printf("struct %v\n", server.fsconf)
	// find address of this server
	var address string
	for i:=0;i<len(server.fsconf);i++{
		if id==server.fsconf[i].Id{
			address=server.fsconf[i].Address
			break

//			fmt.Printf("adress: %v\n", address)
		}
	}
	return address
}

func reply(conn *net.TCPConn, msg *fs.Msg) bool {
	var err error
	write := func(data []byte) {
		if err != nil {
			return
		}
		_, err = conn.Write(data)
	}
	var resp string
	switch msg.Kind {
	case 'C': // read response
		resp = fmt.Sprintf("CONTENTS %d %d %d", msg.Version, msg.Numbytes, msg.Exptime)
	case 'O':
		resp = "OK "
		if msg.Version > 0 {
			resp += strconv.Itoa(msg.Version)
		}
	case 'F':
		resp = "ERR_FILE_NOT_FOUND"
	case 'V':
		resp = "ERR_VERSION " + strconv.Itoa(msg.Version)
	case 'M':
		resp = "ERR_CMD_ERR"
	case 'I':
		resp = "ERR_INTERNAL"
	case 'R':

		resp = "ERR_REDIRECT "+string(msg.Contents[:])

	default:
		fmt.Printf("Unknown response kind '%c'", msg.Kind)
		return false
	}
	resp += "\r\n"
	write([]byte(resp))
	if msg.Kind == 'C' {
		write(msg.Contents)
		write(crlf)
	}
	return err == nil
}

func (server *Server) serve(clientid int64, clientCommitCh chan error, conn *net.TCPConn) {
	
	reader := bufio.NewReader(conn)
	for {
		msg, msgerr, fatalerr := fs.GetMsg(clientid, reader)
//		msg.ClientId=clientid
		if fatalerr != nil {
			reply(conn, &fs.Msg{Kind: 'M'})
			conn.Close()
			break
		}

		if msgerr != nil {
			if (!reply(conn, &fs.Msg{Kind: 'M'})) {
				conn.Close()
				break
			}
			continue
		}

		// if message not of read type
		if msg.Kind == 'w' || msg.Kind=='c' || msg.Kind=='d'{
			dbytes, err := encode(*msg)
			if err != nil {
				if (!reply(conn, &fs.Msg{Kind: 'I'})) { 
					conn.Close()
					break
				}
				continue
			}
			// append the msg to the raft node log 
			server.rn.Append(dbytes)
			//wait for the msg to appear on the client commit channel
			errval := <- clientCommitCh
//			fmt.Printf("Err val: %v , ServerId: %v\n", errval, rc.Id())
			if errval != nil {
				msgContent := server.getAddress(server.rn.LeaderId())
//				fmt.Printf("Leader address : %v\n", rc.LeaderId())
				reply(conn, &fs.Msg{Kind: 'R', Contents: []byte(msgContent)})
				conn.Close()
				break
			}

		}
		response := fs.ProcessMsg(msg)
		if !reply(conn, response) {
			conn.Close()
			break
		}
	}
}



func (server *Server) ListenCommitChannel(){
	getMsg := func(index int) {
		err, emsg := server.rn.Get(index)
		if(err!=nil){
			fmt.Printf("ListenCommitChannel: Error in getting message 4")
			assert(err==nil)
		}
		
		dmsg,err :=decode(emsg)
		if err!=nil{
			fmt.Printf("ListenCommitChannel: Error in decoding message 3")
			assert(err==nil)
		}

		server.ClientChanMap[dmsg.ClientId]<- nil
	}

	var prevLogIndexProcessed = -1
	for{
		//listen on commit channel of raft node 
		commitval := <-server.rn.CommitChannel()
		if(commitval.Err != nil){
			//Redirect the client. Assume the server for which this server voted is the leader. So, redirect it there.
			dmsg,err :=decode(commitval.Data)
			if err!=nil{
				fmt.Printf("ListenCommitChannel: Error in decoding message 1")
				assert(err==nil)
			}
//			server.ClientChanMap[dmsg.ClientId]<-ClientEntry{dmsg,errors.New("ERR_RED")}
			server.ClientChanMap[dmsg.ClientId]<- errors.New("ERR_REDIRECT")

		}else{
			//check if there are missing or duplicate commits 
			if commitval.Index <= prevLogIndexProcessed{
				// already processed. So continue
				continue
			}
			for i := prevLogIndexProcessed + 1; i< commitval.Index; i++{
				getMsg(i)
			}


			dmsg,err :=decode(commitval.Data)
			if err!=nil{
			fmt.Printf("ListenCommitChannel: Error in decoding message 3")
			assert(err==nil)
		}
//			server.ClientChanMap[dmsg.ClientId]<-ClientEntry{dmsg,nil}
			server.ClientChanMap[dmsg.ClientId] <- nil
			prevLogIndexProcessed=commitval.Index	
		}	
		
	}
	
}

func serverMain(id int, conf ClusterConfig) {
	var server Server
	gob.Register(MsgEntry{}) 
	var clientid int64 = 0  
	
	// make map for mapping client id with corresponding receiving clientcommitchannel
	server.ClientChanMap = make(map[int64]chan error)

	// fsconf stores array of the id and addresses of all file servers
	server.fsconf = makeFSNetConfig(conf)
	
	// find address of this server
	address := server.getAddress(id)

	/*var address string
	for i:=0;i<len(server.fsconf);i++{
		if id==server.fsconf[i].Id{
			address=server.fsconf[i].Address
		}
	}*/
	// start the file server 
	tcpaddr, err := net.ResolveTCPAddr("tcp", address) 
	check(err)
	tcp_acceptor, err := net.ListenTCP("tcp", tcpaddr)
	check(err)

	// make raft server object
	raftconf := makeRaftNetConfig(conf)	
	server.rn = BringNodeUp(id,raftconf)
	// start listening on raft commit channel
	go server.ListenCommitChannel()
	// start raft server to process events
	go server.rn.processEvents()

	// start accepting connection from clients
	for {
		tcp_conn, err := tcp_acceptor.AcceptTCP()
		check(err)

		// assign id and commit chan to client
		clientid=(clientid+1)%MAX_CLIENTS
		clientCommitCh := make(chan error)
		server.ClientChanMap[clientid]=clientCommitCh

		// go and serve the client connection
		go server.serve(clientid, clientCommitCh, tcp_conn)
	}
}
/*
func main() {
	serverMain()
}*/
