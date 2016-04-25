package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func clearFiles(file string) {
	rmlog(file)
}

var fileServer []*exec.Cmd

func StartServer(i int, restartflag string) {
	fileServer[i-1] = exec.Command("./assign4", strconv.Itoa(i), restartflag)
	fileServer[i-1].Stdout = os.Stdout
	fileServer[i-1].Stdin = os.Stdin
	fileServer[i-1].Start()
}

func StartAllServerProcess() {
	conf := readJSONFile("Config.json")
	fileServer = make([]*exec.Cmd, len(conf.Peers))
	for i := 1; i <= len(conf.Peers); i++ {
		clearFiles("myLogDir" + strconv.Itoa(i) + "/logfile")
		clearFiles("myStateDir" + strconv.Itoa(i) + "/mystate")
		StartServer(i, "false")
	}
	time.Sleep(7 * time.Second)
}

func KillServer(i int) {

	if err := fileServer[i-1].Process.Kill(); err != nil {
		log.Fatal("failed to kill server: ", err)
	}

}

func KillAllServerProcess() {
	conf := readJSONFile("Config.json")
	for i := 1; i <= len(conf.Peers); i++ {
		KillServer(i)
		clearFiles("myLogDir" + strconv.Itoa(i) + "/logfile")
		clearFiles("myStateDir" + strconv.Itoa(i) + "/mystate")
	}
	time.Sleep(2 * time.Second)
}

func Redirect(t *testing.T, m *Msg, cl *Client) (c *Client) {
	redirectAddress := string(m.Contents)
	cl.close()
	cl = mkClient(t, redirectAddress)
	return cl
}

// ############################################## Basic Test #####################################################################

//This testcase test that all servers should start functioning properly. Clients try to connect to each of the servers for update
//operations. But ultimately they should be redirected to the leader. After all the updates, a client try to read the updated files
// from one of the follower. This should be successful. Finally, all the servers are killed.

func TestBasic(t *testing.T) {

	StartAllServerProcess()

	var leader string

	// make client connection to all the servers
	nclients := 5
	clients := make([]*Client, nclients)
	addr := [5]string{"localhost:9001", "localhost:9002", "localhost:9003", "localhost:9004", "localhost:9005"}
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, addr[i%5])
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}

	// Try a write command at each server. All the clients except the one connected to leader should be redirected
	for i := 0; i < nclients; i++ {
		str := "Distributed System is a zoo"
		filename := fmt.Sprintf("DSsystem%v", i)
		cl := clients[i]
		m, err := cl.write(filename, str, 0)
		for err == nil && m.Kind == 'R' {
			leader = string(m.Contents)
			cl = Redirect(t, m, cl)
			clients[i] = cl
			m, err = cl.write(filename, str, 0)
		}
		expect(t, m, &Msg{Kind: 'O'}, "Test_Basic: Write success", err)
	}

	// Sleep for some time and let the entries be replicated at followers
	time.Sleep(3 * time.Second)

	//make a client connection to follower and try to read
	leaderId := leader[len(leader)-1:]
	l, _ := strconv.Atoi(leaderId)
	var foll int
	if l == 5 {
		foll = 1
	} else {
		foll = l + 1
	}

	cl := mkClient(t, "localhost:900"+strconv.Itoa(foll))
	for i := 0; i < nclients; i++ {
		filename := fmt.Sprintf("DSsystem%v", i)
		m, err := cl.read(filename)
		expect(t, m, &Msg{Kind: 'C', Contents: []byte("Distributed System is a zoo")}, "TestBasic: Read fille", err)
	}

	//	KillAllServerProcess()
	fmt.Println("TestBasic : Pass")
}

//########################################### Assignment 1 Basic Test Cases #########################################################//

func TestRPC_BasicSequential(t *testing.T) {
	//	StartAllServerProcess()

	cl := mkClient(t, "localhost:9001")

	// Read non-existent file cs733net
	m, err := cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Read non-existent file cs733net
	m, err = cl.delete("cs733net")
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.delete("cs733net")
	}
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Write file cs733net
	data := "Cloud fun"
	m, err = cl.write("cs733net", data, 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.delete("cs733net")

	}
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)

	// CAS in new value
	version1 := m.Version
	data2 := "Cloud fun 2"
	// Cas new value
	m, err = cl.cas("cs733net", version1, data2, 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.cas("cs733net", version1, data2, 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "cas success", err)

	// Expect to read it back
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "read my cas", err)

	// Expect Cas to fail with old version
	m, err = cl.cas("cs733net", version1, data, 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.cas("cs733net", version1, data, 0)
	}
	expect(t, m, &Msg{Kind: 'V'}, "cas version mismatch", err)

	// Expect a failed cas to not have succeeded. Read should return data2.
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data2)}, "failed cas to not have succeeded", err)

	// delete
	m, err = cl.delete("cs733net")
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.delete("cs733net")
	}
	expect(t, m, &Msg{Kind: 'O'}, "delete success", err)

	// Expect to not find the file
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	cl.close()

	//	KillAllServerProcess()

	fmt.Println("TestRPC_BasicSequential : Pass")
}

func TestRPC_Binary(t *testing.T) {
	//	StartAllServerProcess()
	cl := mkClient(t, "localhost:9003")
	defer cl.close()
	// Write binary contents
	data := "\x00\x01\r\n\x03" // some non-ascii, some crlf chars
	m, err := cl.write("binfile", data, 0)

	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("binfile", data, 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back
	m, err = cl.read("binfile")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(data)}, "read my write", err)

	//	KillAllServerProcess()
	fmt.Println("TestRPC_Binary : Pass")
}

func TestRPC_BasicTimer(t *testing.T) {
	//	StartAllServerProcess()
	cl := mkClient(t, "localhost:9003")
	defer cl.close()

	// Write file cs733, with expiry time of 2 seconds
	str := "Cloud fun"
	m, err := cl.write("cs733", str, 2)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cs733", str, 2)
	}
	expect(t, m, &Msg{Kind: 'O'}, "write success", err)

	// Expect to read it back immediately.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "read my cas", err)

	time.Sleep(3 * time.Second)
	// Expect to not find the file after expiry
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Recreate the file with expiry time of 1 second
	m, err = cl.write("cs733", str, 1)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cs733", str, 1)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)

	// Overwrite the file with expiry time of 4. This should be the new time.
	m, err = cl.write("cs733", str, 4)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cs733", str, 4)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file overwriten with exptime=4", err)

	// The last expiry time was 3 seconds. We should expect the file to still be around 2 seconds later
	time.Sleep(2 * time.Second)
	// Expect the file to not have expired.
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte(str)}, "file to not expire until 4 sec", err)

	time.Sleep(3 * time.Second)
	// 5 seconds since the last write. Expect the file to have expired
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'F'}, "file not found after 4 sec", err)

	// Create the file with an expiry time of 10 sec. We're going to delete it
	// then immediately create it. The new file better not get deleted.
	m, err = cl.write("cs733", str, 10)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cs733", str, 10)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file created for delete", err)

	m, err = cl.delete("cs733")
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.delete("cs733")
	}
	expect(t, m, &Msg{Kind: 'O'}, "deleted ok", err)

	m, err = cl.write("cs733", str, 0) // No expiry
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cs733", str, 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "file recreated", err)

	time.Sleep(1100 * time.Millisecond) // A little more than 1 sec
	m, err = cl.read("cs733")
	expect(t, m, &Msg{Kind: 'C'}, "file should not be deleted", err)

	//	KillAllServerProcess()
	fmt.Println("TestRPC_BasicTimer : Pass")
}

//############################################################# Assignment 1 Concurrent Test Cases ###################################

// nclients write to the same file. At the end the file should be any one clients' last write.

func TestRPC_ConcurrentWrites(t *testing.T) {
	//	StartAllServerProcess()
	nclients := 10 // It works for 500 clients and 5 iterations but takes time
	niters := 5
	clients := make([]*Client, nclients)
	addr := [5]string{"localhost:9001", "localhost:9002", "localhost:9003", "localhost:9004", "localhost:9005"}
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, addr[i%5])
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}
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
				for err == nil && m.Kind == 'R' {
					cl = Redirect(t, m, cl)
					clients[i] = cl
					m, err = cl.write("concWrite", str, 0)
				}
				//			fmt.Printf("write cl %v %v successful \n",i,j)
				if err != nil {
					errCh <- err
					break
				} else {
					ch <- m
				}
			}
		}(i, clients[i])
	}
	time.Sleep(100 * time.Millisecond) // give goroutines a chance
	sem.Done()                         // Go!

	// There should be no errors
	for i := 0; i < nclients*niters; i++ {
		select {
		case m := <-ch:
			if m.Kind != 'O' {
				t.Fatalf("Concurrent write failed with kind=%c", m.Kind)
			}
		case err := <-errCh:
			t.Fatal(err)
		}
	}
	m, _ := clients[0].read("concWrite")

	// Ensure the contents are of the form "cl <i> 4"
	// The last write of any client ends with " 4"
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), "4")) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg = %v", m)
	}

	//	KillAllServerProcess()
	fmt.Println("TestRPC_ConcurrentWrites : Pass")
}

// nclients cas to the same file. At the end the file should be any one clients' last write.
// The only difference between this test and the ConcurrentWrite test above is that each
// client loops around until each CAS succeeds. The number of concurrent clients has been
// reduced to keep the testing time within limits.
func TestRPC_ConcurrentCas(t *testing.T) {
	//	StartAllServerProcess()
	nclients := 5
	niters := 2
	addr := [5]string{"localhost:9001", "localhost:9002", "localhost:9003", "localhost:9004", "localhost:9005"}
	clients := make([]*Client, nclients)
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, addr[i%5])
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}

	var sem sync.WaitGroup // Used as a semaphore to coordinate goroutines to *begin* concurrently
	sem.Add(1)

	m, err := clients[0].write("concCas", "first", 0)
	for err == nil && m.Kind == 'R' {
		clients[0] = Redirect(t, m, clients[0])
		m, err = clients[0].write("concCas", "first", 0)
	}
	ver := m.Version
	if m.Kind != 'O' || ver == 0 {
		t.Fatalf("Expected write to succeed and return version")
	}

	var wg sync.WaitGroup
	wg.Add(nclients)

	errorCh := make(chan error, nclients)

	for i := 0; i < nclients; i++ {
		go func(i int, ver int, cl *Client) {
			sem.Wait()
			defer wg.Done()
			for j := 0; j < niters; j++ {
				str := fmt.Sprintf("cl %d %d", i, j)
				for {
					m, err := cl.cas("concCas", ver, str, 0)
					for err == nil && m.Kind == 'R' {
						cl = Redirect(t, m, cl)
						clients[i] = cl
						m, err = cl.cas("concCas", ver, str, 0)
					}

					if err != nil {
						errorCh <- err
						return
					} else if m.Kind == 'O' {
						//						fmt.Printf("cas cl %v %v successful \n",i,j)
						break
					} else if m.Kind != 'V' {
						errorCh <- errors.New(fmt.Sprintf("Expected 'V' msg, got %c", m.Kind))
						return
					}
					ver = m.Version // retry with latest version
				}
			}
		}(i, ver, clients[i])
	}

	time.Sleep(100 * time.Millisecond) // give goroutines a chance
	sem.Done()                         // Start goroutines
	wg.Wait()                          // Wait for them to finish
	select {
	case e := <-errorCh:
		t.Fatalf("Error received while doing cas: %v", e)
	default: // no errors
	}
	m, _ = clients[0].read("concCas")
	if !(m.Kind == 'C' && strings.HasSuffix(string(m.Contents), " 1")) {
		t.Fatalf("Expected to be able to read after 1000 writes. Got msg.Kind = %d, msg.Contents=%s", m.Kind, m.Contents)
	}

	//	KillAllServerProcess()
	fmt.Println("TestRPC_ConcurrentCas : Pass")
}

// ################################################## Node Failure Test #####################################

// In this test case, we do concurrent append at the leader and after recieving all the responses,
// we kill one of the follower and issue some writes and deletes to the leader. After that start the
// follower again. After some time, we expect the commands issued to the leader
// to be applied on the follower as well

func Test_FollowerFailure(t *testing.T) {
	//	StartAllServerProcess()
	nclients := 10
	var leader string

	//make 5 clients and make connections to the server
	clients := make([]*Client, nclients)
	addr := [5]string{"localhost:9001", "localhost:9002", "localhost:9003", "localhost:9004", "localhost:9005"}
	for i := 0; i < nclients; i++ {
		cl := mkClient(t, addr[i%5])
		if cl == nil {
			t.Fatalf("Unable to create client #%d", i)
		}
		defer cl.close()
		clients[i] = cl
	}
	done := make(chan bool)
	// 10 Clients simultaneously attempt to write to a file
	var sem sync.WaitGroup // Used as a semaphore to coordinate goroutines to begin concurrently
	sem.Add(1)
	for i := 0; i < nclients; i++ {
		go func(i int, cl *Client) {
			sem.Wait()
			str := fmt.Sprintf("cl %d", i)
			m, err := cl.write("concWrite", str, 0)
			for err == nil && m.Kind == 'R' {
				leader = string(m.Contents)
				cl = Redirect(t, m, cl)
				clients[i] = cl
				m, err = cl.write("concWrite", str, 0)
			}
			done <- true
		}(i, clients[i])

	}
	time.Sleep(100 * time.Millisecond) // give goroutines a chance
	sem.Done()
	for i := 1; i <= nclients; i++ {
		<-done
	}
	//As soon as all the responses are received, kill a follower Node.
	// Note : Some of the appends may not have been received at this follower
	leaderId := leader[len(leader)-1:]
	l, _ := strconv.Atoi(leaderId)
	var foll int
	if l == 5 {
		foll = 1
	} else {
		foll = l + 1
	}

	KillServer(foll)

	time.Sleep(500 * time.Millisecond)

	// Make a client connect to the leader and create 2 new files and delete the previous file
	cl := mkClient(t, leader)

	m, err := cl.write("cloudSuspense", "zoo", 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cloudSuspense", "zoo", 0)
	}

	data := "Cloud fun"
	m, err = cl.write("cs733net", data, 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cs733net", data, 0)

	}

	m, err = cl.delete("concWrite")
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.delete("concWrite")
	}

	time.Sleep(500 * time.Millisecond)

	// Bring back the follower
	StartServer(foll, "true")
	time.Sleep(4 * time.Second) // wait for some time for the entries to be replicated and then read the three files

	follower := "localhost:900" + strconv.Itoa(foll)
	// Make a client connection to the follower and start reading
	cl = mkClient(t, follower)

	m, err = cl.read("cloudSuspense")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("zoo")}, "read my write", err)

	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("Cloud fun")}, "read cloud fun", err)

	m, err = cl.read("concWrite")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	KillAllServerProcess()
	fmt.Println("Test_FollowerFailure : Pass")

}

//In this test case, we do a few writes on the leader, then kill the leader.
//Wait for the new leader to be elected and then do a few writes on the new leader.
//After sometime, we bring the previous leader back and waited for sometime for
//entries to be replicated on the previous leader and then read the modified files.

func Test_LeaderFailure(t *testing.T) {
	StartAllServerProcess()

	var leader string = "localhost:9001"

	cl := mkClient(t, "localhost:9001")

	m, err := cl.write("cloudSuspense", "zoo", 0)
	for err == nil && m.Kind == 'R' {
		leader = string(m.Contents)
		cl = Redirect(t, m, cl)
		m, err = cl.write("cloudSuspense", "zoo", 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderFailure : cloudSuspensesuccess", err)

	data := "Cloud fun"
	m, err = cl.write("cs733net", data, 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("cs733net", data, 0)

	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderFailure : cloudFun suspense", err)

	//Kill leader
	leaderId := leader[len(leader)-1:]
	l, _ := strconv.Atoi(leaderId)
	KillServer(l)

	// wait for few seconds for one of the node to timeout and a new leader to get elected
	time.Sleep(10 * time.Second)

	if l == 1 {
		cl = mkClient(t, "localhost:9002")
	} else {
		cl = mkClient(t, "localhost:9001")
	}

	// Now issue another command
	m, err = cl.delete("cs733net")
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.delete("cs733net")
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderFailure : Deletecs733success", err)

	m, err = cl.write("DSystems", "zoo", 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		m, err = cl.write("Dsystems", "zoo", 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderFailure : DSystemsSuccess", err)

	// Bring back the previous leader
	StartServer(l, "true")
	time.Sleep(10 * time.Second)
	// make client connection with the previous leader and read all the above changes
	cl = mkClient(t, leader)

	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	m, err = cl.read("DSystems")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("zoo")}, "read my write", err)

	m, err = cl.read("cloudSuspense")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("zoo")}, "read my write", err)

	KillAllServerProcess()
	fmt.Println("Test_LeaderFailure : Pass")
}

// ########################################## Testing multiple leader failures ##################################################

// In this test case, we kill 3 consecutive leaders and check if at the end all the changes committed
// by each of the leader are reflected in the file system.

func Test_LeaderMultipleFailure(t *testing.T) {

	StartAllServerProcess()

	var leader string = "localhost:9001"

	// All nodes are working system should work
	cl := mkClient(t, "localhost:9001")

	m, err := cl.write("cloudSuspense", "zoo", 0)
	for err == nil && m.Kind == 'R' {
		leader = string(m.Contents)
		cl = Redirect(t, m, cl)
		//		fmt.Printf("redirected to: %v\n", string(m.Contents))
		m, err = cl.write("cloudSuspense", "zoo", 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderMultipleFailure: Write CloudSuspense success", err)

	m, err = cl.read("cloudSuspense")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("zoo")}, "Test_LeaderMultipleFailure: cloudSuspense read 0", err)

	//Kill the leader
	leaderId := leader[len(leader)-1:]
	l, _ := strconv.Atoi(leaderId)
	KillServer(l)

	previousLeader := l

	// wait for few seconds for one of the node to timeout and a new leader to get elected
	time.Sleep(10 * time.Second)

	//system should still work
	if l == 1 {
		cl = mkClient(t, "localhost:9002")
		leader = "localhost:9002"
	} else {
		cl = mkClient(t, "localhost:9001")
		leader = "localhost:9001"
	}

	m, err = cl.write("Dsystems", "zoo", 0)
	for err == nil && m.Kind == 'R' {
		leader = string(m.Contents)
		cl = Redirect(t, m, cl)
		//		fmt.Printf("redirected to: %v\n", string(m.Contents))
		m, err = cl.write("Dsystems", "zoo", 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderMultipleFailure : DSystemsSuccess", err)

	m, err = cl.read("Dsystems")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("zoo")}, "Test_LeaderMultipleFailure: DSystems read 0", err)

	// Now kill the leader again
	leaderId = leader[len(leader)-1:]
	l, _ = strconv.Atoi(leaderId)
	KillServer(l)

	// wait for few seconds for one of the node to timeout and a new leader to get elected.
	time.Sleep(10 * time.Second)

	//system should still work
	if l != 1 && previousLeader != 1 {
		cl = mkClient(t, "localhost:9001")
		leader = "localhost:9001"
	} else if l != 2 && previousLeader != 2 {
		cl = mkClient(t, "localhost:9002")
		leader = "localhost:9002"
	} else if l != 3 && previousLeader != 3 {
		cl = mkClient(t, "localhost:9003")
		leader = "localhost:9003"
	}

	m, err = cl.delete("cloudSuspense")
	for err == nil && m.Kind == 'R' {
		leader = string(m.Contents)
		cl = Redirect(t, m, cl)
		//		fmt.Printf("redirected to: %v\n", string(m.Contents))
		m, err = cl.delete("cloudSuspense")
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderMultipleFailure : DeletecloudSuspensesuccess", err)

	data := "Cloud fun"
	m, err = cl.write("cs733net", data, 0)
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		//		fmt.Printf("redirected to: %v\n", string(m.Contents))
		m, err = cl.write("cs733net", data, 0)

	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderMultipleFailure : write cs733 success", err)

	// Bring back the first node killed
	StartServer(previousLeader, "true")

	time.Sleep(5 * time.Second)

	previousLeader = l

	// Kill the leader again
	leaderId = leader[len(leader)-1:]
	l, _ = strconv.Atoi(leaderId)
	KillServer(l)

	// wait for few seconds for one of the node to timeout and a new leader to get elected
	time.Sleep(10 * time.Second)

	//system should still work
	if l != 1 && previousLeader != 1 {
		cl = mkClient(t, "localhost:9001")
		leader = "localhost:9001"
	} else if l != 2 && previousLeader != 2 {
		cl = mkClient(t, "localhost:9002")
		leader = "localhost:9002"
	} else if l != 3 && previousLeader != 3 {
		cl = mkClient(t, "localhost:9003")
		leader = "localhost:9003"
	}

	m, err = cl.delete("Dsystems")
	for err == nil && m.Kind == 'R' {
		cl = Redirect(t, m, cl)
		//		fmt.Printf("redirected to: %v\n", string(m.Contents))
		m, err = cl.delete("Dsystems")
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderMultipleFailure : DeleteDSystemssuccess", err)

	// Bring back the previous leader
	StartServer(previousLeader, "true")
	time.Sleep(10 * time.Second)

	follower := "localhost:900" + strconv.Itoa(previousLeader)
	// make client connection with the follower and read all the above changes
	cl = mkClient(t, follower)

	// check the status of files modified by previous servers and confirm if they are as expected
	m, err = cl.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("Cloud fun")}, "Test_LeaderMultipleFailure: Cs733 read", err)

	m, err = cl.read("Dsystems")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)
	//	expect(t, m, &Msg{Kind: 'C', Contents: []byte("zoo")}, "Test_LeaderMultipleFailure: DSystems read", err)

	m, err = cl.read("cloudSuspense")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	KillAllServerProcess()
	fmt.Println("Test_LeaderMultipleFailure : Pass")
}

//################################### What if minority of nodes are alive ? ################################################

func Test_MajorityNodeFailure(t *testing.T) {

	StartAllServerProcess()

	var leader string = "localhost:9001"

	// All nodes are working system should work
	cl := mkClient(t, "localhost:9001")

	m, err := cl.write("cloudSuspense", "zoo", 0)
	for err == nil && m.Kind == 'R' {
		leader = string(m.Contents)
		cl = Redirect(t, m, cl)
		m, err = cl.write("cloudSuspense", "zoo", 0)
	}
	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderMultipleFailure: Write CloudSuspense success", err)
	time.Sleep(1 * time.Second)

	// Kill 3 followers .
	leaderId := leader[len(leader)-1:]
	l, _ := strconv.Atoi(leaderId)

	for i := 1; i <= 3; i++ {
		KillServer((l+i)%5 + 1)
	}
	// give an append to leader. The client will hang and wait for response and leader will indefinitely try to get votes from majority
	var sem sync.WaitGroup
	sem.Add(1)
	go func() {
		data := "Cloud fun"
		m, err = cl.write("cs733net", data, 0)
		for err == nil && m.Kind == 'R' {
			cl = Redirect(t, m, cl)
			m, err = cl.write("cs733net", data, 0)

		}
		//	expect(t, m, &Msg{Kind: 'O'}, "Test_LeaderMultipleFailure : write cs733 success", err)
		sem.Done()
	}()

	time.Sleep(5 * time.Second) // client is stuck for this period

	cl1 := mkClient(t, leader)
	m, err = cl1.read("cs733net")
	expect(t, m, &Msg{Kind: 'F'}, "file not found", err)

	// Now start servers
	for i := 1; i <= 3; i++ {
		StartServer((l+i)%5+1, "true")
	}

	sem.Wait()

	m, err = cl1.read("cs733net")
	expect(t, m, &Msg{Kind: 'C', Contents: []byte("Cloud fun")}, "file not found", err)

	KillAllServerProcess()
	fmt.Println("Test_MajorityNodeFailure : Pass")
}

// ################################################### Utility functions #####################################################

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
			if fieldNum >= len(fields) {
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
		if len(fields) < 2 {
			msg.Contents = []byte("localhost:9005")
		} else {
			msg.Contents = []byte(fields[1])
		}
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
		fmt.Printf("Error in mkclient\n")
		t.Fatal(err)
	}
	return client
}

func expect(t *testing.T, response *Msg, expected *Msg, errstr string, err error) {
	if err != nil {
		KillAllServerProcess()
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
		KillAllServerProcess()
		t.Fatal("Expected " + errstr)
	}
}
