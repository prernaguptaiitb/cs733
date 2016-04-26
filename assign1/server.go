package main

import (
	"fmt"
	"net"
	"log"
	"bufio"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// File Object
type FileType struct {
	filename       string
	version        int64
	expTime        int64
	numBytes       int
	isExpTimeGiven bool
	contentBuf     []byte
}

// map to retrieve file from filename
var filemap = make(map[string]FileType)
var tick int64 = 0
var mapLock sync.RWMutex

func Timer() {
	ticker := time.NewTicker(time.Second * 1)
	go func() {
		for _ = range ticker.C {
			atomic.AddInt64(&tick, int64(1))
		}
	}()
}

func ErrorInvalidCmd(cmd string, Connect net.Conn) {
	var msg_send string = "ERR_CMD_ERR\r\n"
	Connect.Write([]byte(msg_send))
}

func WriteFile(Connect net.Conn, reader *bufio.Reader, filename string, noBytes string, expiryTime string) {
	var isexpired bool
	var expTime int
	numBytes, err1 := strconv.Atoi(noBytes)
	if err1 != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
	}
	if expiryTime == "NIL" {
		expTime = 0
		isexpired = false
	} else {
		var err2 error
		expTime, err2 = strconv.Atoi(expiryTime)
		isexpired = true
		if err2 != nil {
			Connect.Write([]byte("ERR_CMD_ERR\r\n"))
			return
		}
	}
	buf := make([]byte, numBytes)

	c := make(chan bool)
	go func() {
		select {
		case <-time.After(20 * time.Second):
			Connect.Write([]byte("ERR_INTERNAL\r\n"))
			Connect.Close()
			return
		case <-c:
			//				fmt.Println("Successful\r\n")
		}
	}()
	_, err := io.ReadFull(reader, buf)
	if err != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
	}
	n, err := reader.ReadByte()
	if err != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}

	if n != byte('\r') {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}
	m, err := reader.ReadByte()
	if err != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}

	if m != byte('\n') {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}
	c <- true
	mapLock.Lock()
	file, ok := filemap[filename]
	// check if file already present in map
	if ok == false {
		newfile := new(FileType)
		newfile.filename = filename
		newfile.version = 1
		if isexpired == true {
			newfile.isExpTimeGiven = true
			newfile.expTime = tick + int64(expTime)
		} else {
			newfile.isExpTimeGiven = false
			newfile.expTime = int64(expTime)
		}

		newfile.numBytes = numBytes
		newfile.contentBuf = buf
		filemap[filename] = *(newfile)
		mapLock.Unlock()
		Connect.Write([]byte("OK " + fmt.Sprintf("%v", newfile.version) + "\r\n"))

	} else {
		file.version += 1
		file.numBytes = numBytes
		file.contentBuf = buf
		if isexpired == true {
			file.isExpTimeGiven = true
			file.expTime = int64(expTime) + tick
		} else {
			file.isExpTimeGiven = false
			file.expTime = int64(expTime)
		}
		filemap[filename] = file
		mapLock.Unlock()
		Connect.Write([]byte("OK " + fmt.Sprintf("%v", file.version) + "\r\n"))
	}

}

func readFile(Connect net.Conn, filename string) {
	mapLock.RLock()
	file, ok := filemap[filename]
	mapLock.RUnlock()
	if ok == false {
		Connect.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
		return
	} else {
		if file.isExpTimeGiven == false {
			Connect.Write([]byte(fmt.Sprintf("CONTENTS %v %v \r\n", file.version, file.numBytes)))
			Connect.Write(append(file.contentBuf, []byte("\r\n")...))
		} else {
			if file.expTime < tick {
				Connect.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
				return
			} else {
				Connect.Write([]byte(fmt.Sprintf("CONTENTS %v %v %v \r\n", file.version, file.numBytes, int64(file.expTime)-tick)))
				Connect.Write(append(file.contentBuf, []byte("\r\n")...))
			}
		}
	}
}

func compareAndSwap(Connect net.Conn, reader *bufio.Reader, filename string, version string, noBytes string, expiryTime string) {
	numBytes, err := strconv.Atoi(noBytes)
	if err != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}
	buf := make([]byte, numBytes)
	c := make(chan bool)
	go func() {
		select {
		case <-time.After(60 * time.Second):
			Connect.Write([]byte("ERR_INTERNAL\r\n"))
			Connect.Close()
			return
		case <-c:
		}
	}()
	_, err1 := io.ReadFull(reader, buf)
	if err1 != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}
	n, err := reader.ReadByte()
	if err != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}

	if n != byte('\r') {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}
	m, err := reader.ReadByte()
	if err != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}

	if m != byte('\n') {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}
	var isexpired bool
	var expTime int
	ver, err := strconv.Atoi(version)
	if err != nil {
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
	}
	if expiryTime == "NIL" {
		expTime = 0
		isexpired = false
	} else {
		var err2 error
		expTime, err2 = strconv.Atoi(expiryTime)
		isexpired = true
		if err2 != nil {
			Connect.Write([]byte("ERR_CMD_ERR\r\n"))
			return
		}
	}
	mapLock.Lock()
	file, ok := filemap[filename]
	if ok == false {
		mapLock.Unlock()
		Connect.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
	} else {
		if file.isExpTimeGiven == true && file.expTime < tick {
			mapLock.Unlock()
			Connect.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
			return

		}
		versionNum := file.version
		if file.version == int64(ver) {
			file.version += 1
			file.numBytes = numBytes
			file.contentBuf = buf
			if isexpired == true {
				file.isExpTimeGiven = true
				file.expTime = int64(expTime) + tick
			} else {
				file.isExpTimeGiven = false
				file.expTime = int64(expTime)
			}
			filemap[filename] = file
			mapLock.Unlock()
			Connect.Write([]byte("OK " + fmt.Sprintf("%v", versionNum+1) + "\r\n"))
		} else {
			versionnum := file.version
			mapLock.Unlock()
			Connect.Write([]byte("ERR_VERSION " + fmt.Sprintf("%v", versionnum) + " \r\n"))
			return
		}
	}
}

func deleteFile(Connect net.Conn, filename string) {
	mapLock.Lock()
	_, ok := filemap[filename]
	if ok == false {
		mapLock.Unlock()
		Connect.Write([]byte("ERR_FILE_NOT_FOUND\r\n"))
	} else {
		delete(filemap, filename)
		mapLock.Unlock()
		Connect.Write([]byte("OK\r\n"))
	}
}

func IsValidCmd(InpCommand string, Connect net.Conn, reader *bufio.Reader) {
	tokenizedCmd := strings.Fields(InpCommand)
	l := len(tokenizedCmd)
	if l == 0 {
		ErrorInvalidCmd("default", Connect)
	} else if tokenizedCmd[0] == "write" {
		if l == 3 {
			WriteFile(Connect, reader, tokenizedCmd[1], tokenizedCmd[2], "NIL")
		} else if l == 4 {
			WriteFile(Connect, reader, tokenizedCmd[1], tokenizedCmd[2], tokenizedCmd[3])
		} else {
			ErrorInvalidCmd("write", Connect)
		}
	} else if tokenizedCmd[0] == "read" {
		if l == 2 {
			readFile(Connect, tokenizedCmd[1])
		} else {
			ErrorInvalidCmd("read", Connect)
		}

	} else if tokenizedCmd[0] == "cas" {
		if l == 4 {
			compareAndSwap(Connect, reader, tokenizedCmd[1], tokenizedCmd[2], tokenizedCmd[3], "NIL")
		} else if l == 5 {
			compareAndSwap(Connect, reader, tokenizedCmd[1], tokenizedCmd[2], tokenizedCmd[3], tokenizedCmd[4])
		} else {
			ErrorInvalidCmd("cas", Connect)
		}

	} else if tokenizedCmd[0] == "delete" {
		if l == 2 {
			deleteFile(Connect, tokenizedCmd[1])
		} else {
			ErrorInvalidCmd("delete", Connect)
		}
	} else {
		//		ErrorInvalidCmd("default", Connect)
		Connect.Write([]byte("ERR_CMD_ERR\r\n"))
		Connect.Close()
		return
	}
}

func handleThisClient(Connect net.Conn) {
	reader := bufio.NewReader(Connect)
	for {
		msg_rec, _, err := reader.ReadLine()
		if err != nil {
	//		log.Println(err)
			Connect.Close()
			return
		}
		IsValidCmd(string(msg_rec), Connect, reader)
	}
}

func serverMain() {
	Timer()
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		//port not free
		log.Println(err)
		return
		//log.Fatal(err)
	}
	for {
		connect, err := listen.Accept()
		if err != nil {
	//		log.Println(err)
			continue
		}
		go handleThisClient(connect)
		//		time.Sleep(time.Second*20)
	}
}

func main() {
	serverMain()
}
