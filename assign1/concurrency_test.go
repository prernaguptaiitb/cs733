package main

import (
	"bufio"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Simple serial check of getting and setting
func TestTCPSimple(t *testing.T) {
	runtime.GOMAXPROCS(1005)
	go serverMain()
	time.Sleep(1 * time.Second) // one second is enough time for the server to start
	name := "hi.txt"
	contents := "bye"
	exptime := 300000
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error()) // report error through testing framework
	}
	scanner := bufio.NewScanner(conn)
	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr := strings.Split(resp, " ") // split into OK and <version>
	expect(t, arr[0], "OK")
	version, err := strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version)) // expect only accepts strings, convert int version to string
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
	scanner.Scan()
	expect(t, contents, scanner.Text())
}

// Test syntax of write command
func TestSyntax(t *testing.T) {
	name := "hi.txt"
	contents := "bye"
	exptime := 300000
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error()) // report error through testing framework
	}
	scanner := bufio.NewScanner(conn)
	// Write a file
	fmt.Fprintf(conn, "write ab %v %v\r\n%v\r\n", name, exptime, contents)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr := strings.Split(resp, " ") // split into OK and <version>
	expect(t, arr[0], "ERR_CMD_ERR")
	conn, err = net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error()) // report error through testing frame
	}
	scanner = bufio.NewScanner(conn)
	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	expect(t, contents, scanner.Text())

}

//Test delete - write a file, read it, delete it, then attempt to read it again

func TestDelete(t *testing.T) {
	name := "f2.txt"
	contents := "bye\r\nprerna"
	exptime := 300
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error()) // report error through testing framework
	}
	scanner := bufio.NewScanner(conn)
	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr := strings.Split(resp, " ") // split into OK and <version>
	expect(t, arr[0], "OK")
	version, err := strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	expect(t, arr[1], fmt.Sprintf("%v", version)) // expect only accepts strings, convert int version to string
	expect(t, arr[2], fmt.Sprintf("%v", len(contents)))
	scanner.Scan()
	expect(t, "bye", scanner.Text())
	scanner.Scan()
	expect(t, "prerna", scanner.Text())
	fmt.Fprintf(conn, "delete %v\r\n", name)
	scanner.Scan()        // read first line
	resp = scanner.Text() // extract the text from the buffer
	//	arr = strings.Split(resp, " ") // split into OK and <version>
	expect(t, resp, "OK")
	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	expect(t, scanner.Text(), "ERR_FILE_NOT_FOUND")
}

// Test behavior of system with 10 clients writing simultaneously to a file
func TestWriteConcurrency(t *testing.T) {
	name := "f1.txt"
	ch := make(chan bool)
	for i := 1; i <= 10; i++ {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			t.Error(err.Error()) // report error through testing framework
		}
		scanner := bufio.NewScanner(conn)
		go writeClient(conn, scanner, i, t, ch)
	}
	//	time.Sleep(1 * time.Second)
	for i := 1; i <= 10; i++ {
		<-ch
	}
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error()) // report error through testing framework
		//		fmt.Println("Here")
	}
	scanner := bufio.NewScanner(conn)
	fmt.Fprintf(conn, "read %v\r\n", name) // try a read now
	scanner.Scan()
	arr := strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	expectOptions(t, scanner.Text())
}

// Test behavior of cas and write together attempted by two clients
func TestCasWrite(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error())
	}
	scanner := bufio.NewScanner(conn)
	name := "f1.txt"
	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	arr := strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	ver := arr[1]
	ch := make(chan bool)
	go write(t, name, ch)
	go cas(t, name, ver, ch)
	<-ch
	<-ch
	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	expect(t, scanner.Text(), "2")
}

// Test system behavior when 5 clients simultaneously attempt cas to a file 
func TestCasConcurrency(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		//		t.Error(err.Error())
		t.Error("Connection failure")
	}
	scanner := bufio.NewScanner(conn)
	name := "f1.txt"
	val := 1
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(conn, "read %v\r\n", name)
		scanner.Scan()
		arr := strings.Split(scanner.Text(), " ")
		expect(t, arr[0], "CONTENTS")
		scanner.Scan()
		scanner.Text()
		ver := arr[1]
		ch := make(chan bool)
		for x := 1; x <= 5; x++ {
			go casVal(t, name, ver, ch, val)
			val++
		}
		for x := 1; x <= 5; x++ {
			<-ch
		}
	}
	fmt.Fprintf(conn, "read %v\r\n", name)
	scanner.Scan()
	arr := strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	expectConcCas(t, scanner.Text())
}

//Test read concurrency - 3 clients read the same value and then try to update. Their writes are concurrent.
func TestReadConcurrency(t *testing.T) {
	name := "f3.txt"
	contents := "0"
	exptime := 300000
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error()) // report error through testing framework
	}
	scanner := bufio.NewScanner(conn)
	// Write a file
	fmt.Fprintf(conn, "write %v %v %v\r\n%v\r\n", name, len(contents), exptime, contents)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr := strings.Split(resp, " ") // split into OK and <version>
	expect(t, arr[0], "OK")
	_, err = strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	ch := make(chan bool)
	for i := 1; i <= 3; i++ {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			t.Error(err.Error()) // report error through testing framework
		}
		go Readandwrite(conn, i, t, ch)
	}
	// wait for goroutines to finish
	for x:=1; x<=3; x++ {
		<-ch
	}
	//attempt read
	fmt.Fprintf(conn, "read %v\r\n", name) // read the file
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	expect(t, "5", scanner.Text())
	for i := 1; i <= 3; i++ {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			t.Error(err.Error()) // report error through testing framework
		}
		go Readandwritediff(conn, i, t, ch)
	}
	// wait for goroutines to finish
	for x:=1; x<=3; x++ {
		<-ch
	}
	//attempt read
	fmt.Fprintf(conn, "read %v\r\n", name) // read the file
	scanner.Scan()
	arr = strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	expectwr(t,  scanner.Text())
}


//------------------------------------------------------------------------------------------------------------------------------
func Readandwrite(conn net.Conn, i int, t *testing.T, ch chan bool){
	name := "f3.txt"
	scanner := bufio.NewScanner(conn)
	fmt.Fprintf(conn, "read %v\r\n", name) // read the file
	scanner.Scan()
	arr := strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	contents:=strings.Split(scanner.Text()," ")
	val,_ := strconv.Atoi(contents[0])
	time.Sleep(1 * time.Second) // sleep for 2 seconds
	//now attempt write
	content:= strconv.Itoa(val+5)
	fmt.Fprintf(conn, "write %v %v\r\n%v\r\n", name, len(content),  content)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr = strings.Split(resp, " ") // split into OK and <version>
	expect(t, arr[0], "OK")
	_, err := strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	ch<-true
}
func Readandwritediff(conn net.Conn, i int, t *testing.T, ch chan bool){
	name := "f3.txt"
	scanner := bufio.NewScanner(conn)
	fmt.Fprintf(conn, "read %v\r\n", name) // read the file
	scanner.Scan()
	arr := strings.Split(scanner.Text(), " ")
	expect(t, arr[0], "CONTENTS")
	scanner.Scan()
	contents:=strings.Split(scanner.Text()," ")
	val,_ := strconv.Atoi(contents[0])
	time.Sleep(1 * time.Second) // sleep for 2 seconds
	//now attempt write
	content:= strconv.Itoa(val+(5*i))
	fmt.Fprintf(conn, "write %v %v\r\n%v\r\n", name, len(content),  content)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr = strings.Split(resp, " ") // split into OK and <version>
	expect(t, arr[0], "OK")
	_, err := strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	ch<-true
}
// Useful testing function
func expect(t *testing.T, a string, b string) {
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}
}
func expectOptions(t *testing.T, a string) {
	check := false
	for i := 101; i <= 110; i++ {
		if strconv.Itoa(i) == a {
			check = true
		}
	}
	if check == false {
		t.Error(fmt.Sprintf("Wrong Output, found %v", a))
	}

}
func expectwr(t *testing.T, a string) {
	check := false
	for i := 10; i <= 20; i+=5 {
		if strconv.Itoa(i) == a {
			check = true
		}
	}
	if check == false {
		t.Error(fmt.Sprintf("Wrong Output, found %v", a))
	}

}

// for cas
func expectcas(t *testing.T, a string) {
	if a != "OK" && a != "ERR_VERSION" {
		t.Error(fmt.Sprintf("Error in Cas"))
	}

}
func expectConcCas(t *testing.T, a string) {
	check := false
	for i := 46; i <= 50; i++ {
		if strconv.Itoa(i) == a {
			check = true
		}
	}
	if check == false {
		t.Error(fmt.Sprintf("Wrong Output, found %v", a))
	}

}
func write(t *testing.T, name string, ch chan bool) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error())
	}
	contents := "2"
	fmt.Fprintf(conn, "write %v %v\r\n%v\r\n", name, len(contents), contents)
	scanner := bufio.NewScanner(conn)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr := strings.Split(resp, " ") // split into OK and <version>
	expect(t, arr[0], "OK")
	_, err = strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	ch <- true
}
func cas(t *testing.T, name string, ver string, ch chan bool) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Error(err.Error())
	}
	contents := "5"
	fmt.Fprintf(conn, "cas %v %v %v\r\n%v\r\n", name, ver, len(contents), contents)
	scanner := bufio.NewScanner(conn)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr := strings.Split(resp, " ") // split into OK and <version>
	expectcas(t, arr[0])
	_, err = strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}
	ch <- true
}

func casVal(t *testing.T, name string, ver string, ch chan bool, val int) {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		//	t.Error(err.Error())
		t.Error("Connection failure")
	}
	contents := strconv.Itoa(val)
	fmt.Fprintf(conn, "cas %v %v %v\r\n%v\r\n", name, ver, len(contents), contents)
	scanner := bufio.NewScanner(conn)
	scanner.Scan()                  // read first line
	resp := scanner.Text()          // extract the text from the buffer
	arr := strings.Split(resp, " ") // split into OK and <version>
	expectcas(t, arr[0])
	_, err = strconv.ParseInt(arr[1], 10, 64) // parse version as number
	if err != nil {
		t.Error("Non-numeric version found")
	}

	//fmt.Printf("casval%v",val)
	ch <- true
}
func writeClient(conn net.Conn, scanner *bufio.Scanner, i int, t *testing.T, ch chan bool) {
	name := "f1.txt"
	for j := i; j <= i+100; j = j + 10 {
		// Write a file
		contents := strconv.Itoa(j)
		fmt.Fprintf(conn, "write %v %v \r\n%v\r\n", name, len(contents), contents)
		scanner.Scan()                  // read first line
		resp := scanner.Text()          // extract the text from the buffer
		arr := strings.Split(resp, " ") // split into OK and <version>
		expect(t, arr[0], "OK")
		_, err := strconv.ParseInt(arr[1], 10, 64) // parse version as number
		if err != nil {
			t.Error("Non-numeric version found")
		}
	}
	ch <- true
}
