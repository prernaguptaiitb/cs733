
## Assignment #1 - File server 

Submitted by: Prerna Gupta</br>
Roll No: 143050021

##Introduction

This assignment is to build a simple file server, with a simple read/write interface. The file contents are stored in memory. The server listens on port 8080. Concurrent operations by multiple clients is taken into consideration.

##Installation Instructions

go get  github.com/prernaguptaiitb/cs733-assign1

Folder contains two files: server.go(containing server code) and concurrency_test.go (containing test cases for the server)


### How to use?
Run the server : "go run server.go" .</br>
Start the client : telnet localhost 8080</br>
You can start multiple clients.</br>

### Commands Support:
* Write: create a file, or update the file’s contents if it already exists.
```
write <filename> <numbytes> [<exptime>]\r\n
 <content bytes>\r\n
```
If expiry time not specified then file does not expires.

The server responds with the following:
OK <version>\r\n


* Read: Given a filename, retrieve the corresponding file:
```
read <filename>\r\n
```

The server responds with the following format if file is present at the server.
CONTENTS <version> <numbytes> <exptime> \r\n
 <content bytes>\r\n  

Here <exptime>is the remaining time in seconds left for the file after which it will expire.
If file is not present at the server, the server responds with ERR_FILE_NOT_FOUND.
 
* Compare and swap (cas): This replaces the old file contents with the new content provided the version is still the same.
```
cas <filename> <version> <numbytes> [<exptime>]\r\n
 <content bytes>\r\n
```
The server responds with the new version if successful :-
OK <version>\r\n
If version does not match, server responds with :-
ERR_VERSION <newversion>\r\n

* Delete file
```
delete <filename>\r\n
```
Server response (if successful)
OK\r\n
</br>

Server return following error messages : </br>

* ERR_VERSION <newversion>\r\n (the contents were not updated because of a
version mismatch. The latest version is returned)
* ERR_FILE_NOT_FOUND\r\n (the filename doesn’t exist)
* ERR_CMD_ERR\r\n (the command is not formatted correctly)
