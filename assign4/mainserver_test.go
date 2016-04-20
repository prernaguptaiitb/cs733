package main

import (
	"errors"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"bufio"
//	"github.com/cs733-iitb/log"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"os"
	"sync"
)

type Peer struct {
	Id      int
	Address string
	FSAddress string
}

type ClusterConfig struct {
	Peers []Peer
}

func IfError(err error, msg string){
	if err != nil {
		fmt.Print("Error:", msg)
		os.Exit(1)
	}
}

func readJSONFile(configFile string) ClusterConfig {
	content, err := ioutil.ReadFile(configFile)
	IfError(err,"Error in Reading Json File")
	var conf ClusterConfig
	err = json.Unmarshal(content, &conf)
	IfError(err,"Error in Unmarshaling Json File")
	return conf
}


func clearFiles(file string) {
	rmlog(file)
}

var errNoConn = errors.New("Connection is closed")

type Msg struct {
	// Kind = the first character of the command. For errors, it
	// is the first letter after "ERR_", ('V' for ERR_VERSION, for
	// example), except for "ERR_CMD_ERR", for which the kind is 'M'
	Kind     byte
	Filename string
	Contents []byte
	Numbytes int
	Exptime  int // expiry time in seconds
	Version  int
}


type Client struct {
	conn   *net.TCPConn
	reader *bufio.Reader // a bufio Reader wrapper over conn
}

func (cl *Client) read(filename string) (*Msg, error) {
	cmd := "read " + filename + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) write(filename string, contents string, exptime int) (*Msg, error) {
	var cmd string
	if exptime == 0 {
		cmd = fmt.Sprintf("write %s %d\r\n", filename, len(contents))
	} else {
		cmd = fmt.Sprintf("write %s %d %d\r\n", filename, len(contents), exptime)
	}
	cmd += contents + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) cas(filename string, version int, contents string, exptime int) (*Msg, error) {
	var cmd string
	if exptime == 0 {
		cmd = fmt.Sprintf("cas %s %d %d\r\n", filename, version, len(contents))
	} else {
		cmd = fmt.Sprintf("cas %s %d %d %d\r\n", filename, version, len(contents), exptime)
	}
	cmd += contents + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) delete(filename string) (*Msg, error) {
	cmd := "delete " + filename + "\r\n"
	return cl.sendRcv(cmd)
}

func (cl *Client) send(str string) error {
	if cl.conn == nil {
		return errNoConn
	}
	_, err := cl.conn.Write([]byte(str))
	if err != nil {
		err = fmt.Errorf("Write error in SendRaw: %v", err)
		cl.conn.Close()
		cl.conn = nil
	}
	return err
}

func (cl *Client) sendRcv(str string) (msg *Msg, err error) {
	if cl.conn == nil {
		return nil, errNoConn
	}
	err = cl.send(str)
	if err == nil {
		msg, err = cl.rcv()
	}
	return msg, err
}
func (cl *Client) rcv() (msg *Msg, err error) {
	// we will assume no errors in server side formatting
	line, err := cl.reader.ReadString('\n')
	if err == nil {
		msg, err = parseFirst(line)
		if err != nil {
			return nil, err
		}
		if msg.Kind == 'C' {
			contents := make([]byte, msg.Numbytes)
			var c byte
			for i := 0; i < msg.Numbytes; i++ {
				if c, err = cl.reader.ReadByte(); err != nil {
					break
				}
				contents[i] = c
			}
			if err == nil {
				msg.Contents = contents
				cl.reader.ReadByte() // \r
				cl.reader.ReadByte() // \n
			}
		}
	}
	if err != nil {
		cl.close()
	}
	return msg, err
}

func (cl *Client) close() {
	if cl != nil && cl.conn != nil {
		cl.conn.Close()
		cl.conn = nil
	}
}

func parseFirst(line string) (msg *Msg, err error) {
//	fmt.Printf("%v",line)
	fields := strings.Fields(line)
	msg = &Msg{}

	// Utility function fieldNum to int
	toInt := func(fieldNum int) int {
		var i int
		if err == nil {
			if fieldNum >=  len(fields) {
				err = errors.New(fmt.Sprintf("Not enough fields. Expected field #%d in %s\n", fieldNum, line))
				return 0
			}
			i, err = strconv.Atoi(fields[fieldNum])
		}
		return i
	}

	if len(fields) == 0 {
		return nil, errors.New("Empty line. The previous command is likely at fault")
	}
	switch fields[0] {
	case "OK": // OK [version]
		msg.Kind = 'O'
		if len(fields) > 1 {
			msg.Version = toInt(1)
		}
	case "CONTENTS": // CONTENTS <version> <numbytes> <exptime> \r\n
		msg.Kind = 'C'
		msg.Version = toInt(1)
		msg.Numbytes = toInt(2)
		msg.Exptime = toInt(3)
	case "ERR_VERSION":
		msg.Kind = 'V'
		msg.Version = toInt(1)
	case "ERR_FILE_NOT_FOUND":
		msg.Kind = 'F'
	case "ERR_CMD_ERR":
		msg.Kind = 'M'
	case "ERR_INTERNAL":
		msg.Kind = 'I'
	case "ERR_REDIRECT":
		msg.Kind = 'R'
		msg.Contents=[]byte(fields[1])
	default:
		err = errors.New("Unknown response " + fields[0])
	}
	if err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}


func mkClient(t *testing.T, addr string) *Client {
	var client *Client
	raddr, err := net.ResolveTCPAddr("tcp", addr) // addr: "localhost:8080"
	if err == nil {
		conn, err := net.DialTCP("tcp", nil, raddr)
		if err == nil {
			client = &Client{conn: conn, reader: bufio.NewReader(conn)}
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func startServers(){
	conf := readJSONFile("Config.json")
	for i:=1; i<=len(conf.Peers);i++{
		ld := "myLogDir" + strconv.Itoa(i)
		clearFiles(ld + "/logfile")
		sd := "myStateDir" + strconv.Itoa(i)
		clearFiles(sd + "/mystate")	
		InitializeState(sd)
		go serverMain(i,conf)
	}
	time.Sleep(2 * time.Second)
}

func expect(t *testing.T, response *Msg, expected *Msg, errstr string, err error) {
	if err != nil {
		t.Fatal("Unexpected error: " + err.Error())
	}
	ok := true
	if response.Kind != expected.Kind {
		ok = false
		errstr += fmt.Sprintf(" Got kind='%c', expected '%c'", response.Kind, expected.Kind)
	}
	if expected.Version > 0 && expected.Version != response.Version {
		ok = false
		errstr += " Version mismatch"
	}
	if response.Kind == 'C' {
		if expected.Contents != nil &&
			bytes.Compare(response.Contents, expected.Contents) != 0 {
			ok = false
		}
	}
	if !ok {
		t.Fatal("Expected " + errstr)
	}
}

func Redirect(t *testing.T, m *Msg, cl *Client) (c *Client){
		redirectAddress := string(m.Contents)
		cl.close()
		cl = mkClient(t, redirectAddress)
		return cl
}

func TestRPCMain(t *testing.T) {
	runtime.GOMAXPROCS(4)
	startServers()
}
/*
func TestRPC_BasicSequential(t *testing.T) {
	cl := mkClient(t, "localhost:9003")

	// Read non-existent file cs733net
	m, err := cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)
	fmt.Println("read(cs733net) Pass")

	// Read non-existent file cs733net
	m, err = cl.delete("cs733net")
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.delete("cs733net")
	}
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)
	fmt.Println("delete(cs733net) Pass")

	// Write file cs733net
	data := "Cloud fun"
	m, err = cl.write("cs733net", data, 0)
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.delete("cs733net")

	}
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)
	fmt.Println("write(cs733net, cloudfun, 0) Pass")

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)
	fmt.Println("read(cs733net) Pass")

	// CAS in new value
	version1 := m.Version
	data2 := "Cloud fun 2"
	// Cas new value
	m, err = cl.cas("cs733net", version1, data2, 0)
	expect(t, m, &Msg{Kind: 'O'}, "cas success", err)
	fmt.Println("cas(cs733net, version1, Cloud fun 2) Pass")

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "read my cas", err)
	fmt.Println("read(cs733net) Pass")

	// Expect Cas to fail with old version
	m, err = cl.cas("cs733net", version1, data, 0)
	expect(t, m, &Msg{Kind: 'V'}, "cas version mismatch", err)
	fmt.Println("cas(cs733net, version1) Pass")


	// Expect a failed cas to not have succeeded. Read should return data2.
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "failed cas to not have succeeded", err)
	fmt.Println("read(cs733net) Pass")

	// delete
	m, err = cl.delete("cs733net")
	expect(t, m, &Msg{Kind: 'O'}, "delete success", err)
	fmt.Println("delete(cs733net) Pass")

	// Expect to not find the file
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err) 
	fmt.Println("read(cs733net) Pass")
	cl.close()

}
func TestRPC_Binary(t *testing.T) {
	cl := mkClient(t, "localhost:9003")
	defer cl.close()
	fmt.Println("Testing : TestRPC_Binary")
	// Write binary contents
	data := "\x00\x01\r\n\x03" // some non-ascii, some crlf chars
	m, err := cl.write("binfile", data, 0)
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.write("binfile", data, 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back
	m, err = cl.read("binfile")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)
	fmt.Println("Pass : TestRPC_Binary")
}

func TestRPC_BasicTimer(t *testing.T) {
	cl := mkClient(t, "localhost:9003")
	defer cl.close()

	// Write file cs733, with expiry time of 2 seconds
	str := "Cloud fun"
	m, err := cl.write("cs733", str, 2)
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.write("cs733", str, 2)
	}
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)
	//fmt.Printf("write success\n")

	// Expect to read it back immediately.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "read my cas", err)
//	fmt.Println("read my cas")
	
	time.Sleep( 3* time.Second)
	// Expect to not find the file after expiry
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)
//	fmt.Println("file not found")

	// Recreate the file with expiry time of 1 second
	m, err = cl.write("cs733", str, 1)
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.write("cs733", str, 1)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)
//	fmt.Println("file recreated")

	// Overwrite the file with expiry time of 4. This should be the new time.
	m, err = cl.write("cs733", str, 3)
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.write("cs733", str, 3)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file overwriten with exptime=4", err)
//	fmt.Println("file overwriten with exptime=4")
	// The last expiry time was 3 seconds. We should expect the file to still be around 2 seconds later
	time.Sleep(2 * time.Second)

	// Expect the file to not have expired.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "file to not expire until 4 sec", err)
//	fmt.Println("file to not expire until 4 sec")

	time.Sleep(3 * time.Second)
	// 5 seconds since the last write. Expect the file to have expired
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found after 4 sec", err)
//	fmt.Println("file not found after 4 sec")
	// Create the file with an expiry time of 8 sec. We're going to delete it
	// then immediately create it. The new file better not get deleted. 
	m, err = cl.write("cs733", str, 10)
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.write("cs733", str, 10)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file created for delete", err)
//	fmt.Println("file created for delete")

	m, err = cl.delete("cs733")
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.delete("cs733")
	}
	expect(t, m, &Msg{Kind: 'O'}, "deleted ok", err)
//	fmt.Println("deleted ok")

	m, err = cl.write("cs733", str, 0) // No expiry
	for err == nil && m.Kind=='R'{
		cl=Redirect(t,m,cl)
		m, err = cl.write("cs733", str, 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)
//	fmt.Println("file recreated")

	time.Sleep(1100 * time.Millisecond) // A little more than 1 sec
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C'}, "file should not be deleted", err)
//	fmt.Println("file should not be deleted")

	fmt.Println("Pass : TestRPC_BasicTimer")

}
*/

func TestRPC_ConcurrentWrites(t *testing.T) {
	nclients := 5
	niters := 2
	clients := make([]*Client, nclients)
	addr := [5]string{"localhost:9001", "localhost:9002","localhost:9003","localhost:9004","localhost:9005"}
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, addr[i%5])
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}
	fmt.Println("TCP Connection made")
	errCh := make(chan error, nclients)
	var sem sync.WaitGroup // Used as a semaphore to coordinate goroutines to begin concurrently
	sem.Add(1)
	ch := make(chan *Msg, nclients*niters) // channel for all replies
	for i := 0; i < nclients; i++ {
		go func(i int, cl *Client) {
			sem.Wait()
			for j := 0; j < niters; j++ {				
				str := fmt.Sprintf("cl %d %d", i, j)
				m, err := cl.write("concWrite", str, 0)
				for err == nil && m.Kind=='R'{
					fmt.Println("In Redirect")
					cl=Redirect(t,m,cl)
					clients[i]=cl
					m, err = cl.write("concWrite", str, 0)
				}
				fmt.Printf("%v %v \n",i,j)
				if err != nil {
					errCh <- err
					break
				} else {
					ch <- m
				}
			}
		}(i, clients[i])
	}
	time.Sleep(10 * time.Second) // give goroutines a chance
	sem.Done()                         // Go!

	// There should be no errors
	for i := 0; i < nclients*niters; i++ {
		select {
		case m := <-ch:
			if m.Kind != 'O' {
				t.Fatalf("Concurrent write failed with kind=%c", m.Kind)
			}
		case err := <- errCh:
			t.Fatal(err)
		}
	}
	m, _ := clients[0].read("concWrite")

	// Ensure the contents are of the form "cl <i> 9"
	// The last write of any client ends with " 9"
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), " 1")) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg = %v", m)
	}
}
