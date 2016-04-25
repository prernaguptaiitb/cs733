# Distributed File Server

It is a distributed fault tolerant file server that supports reading, writing, updating and deleting files. It will continue working until majority of the servers in cluster are up. Clients can connect to any of the server in the cluster to get service. It uses RAFT in the background. You can specify the cluster configuration <server-id, Raft-Server-Address, File-Server-Address > for each server in Config.json file.

The system implemented is not linearizable. "READ" requests are not replicated. So, client may read stale content but eventually all server be in same state as long as majority of nodes are up. This "eventual consistency" works for clients who do not require "strict consistency" but require quick response.

Each file has a version and the API supports a “compare and swap” operation based on the version and can have optional expiry time.



### Installation

######## Note: You need to have golang installed in your system.
```
go get github.com/prernaguptaiitb/cs733/assign4
go test github.com/prernaguptaiitb/cs733/assign4
go build
./assign4 <server-id> <false> 
```
######## Note: First argument is the id of the server and the second argument takes value false or true depending on whether you want to start server in the clean state or bring it up from the state it left earlier. 

Each server in the cluster can be started using the above command.

After the cluster is up, clients can connect to the server. In case of "READ" request, clients will be served from the local server while in case of other requests, client will be redirected to the leader with "ERR_REDIRECT" message. 



### Usage Summary : Run 5 servers on different terminals

```
> ./assign4 1 "false"
> ./assign4 2 "false"
> ./assign4 3 "false"
> ./assign4 4 "false"
> ./assign4 5 "false"

> telnet localhost 9001
	Connected to localhost.
  	Escape character is '^]'
  	read foo
  	ERR_FILE_NOT_FOUND
  	write foo 6
  	abcdef
  	OK 1
  	read foo
  	CONTENTS 1 6 0
  	cas foo 1 6 0
	prerna
	OK 2
	read foo
	CONTENTS 2 6 0
	prerna

```

### Commands
Client can issue the following commands :

##### 1. read
Given a filename, retrieve the corresponding file from the server. Return metadata( version number, num_bytes, expiry time) along with the content of file. If the file is not present on the server, "ERR_FILE_NOT_FOUND" error will be received.

Syntax:
```
	read <file_name>\r\n
```

Response (if successful):

``` CONTENTS <version> <num_bytes> <exptime>\r\n ```<br />
``` <content_bytes>\r\n ```


##### 2. write

Create a file, or update the file’s contents if it already exists. Files are given new version numbers when updated.

Syntax:
```
	write <file_name> <num_bytes> [<exp_time>]\r\n
	<content bytes>\r\n
```

Response:

``` OK <version>\r\n ```

##### 3. compare-and-swap (cas)

This replaces the old file contents with the new content provided the version of file is still same as that in the command. Returns OK and new version of the file if successful, else error in cas and the new version, if unsuccessful.

Syntax:
```
	cas <file_name> <version> <num_bytes> [<exp_time>]\r\n
	<content bytes>\r\n
```

Response(If successful): 

``` OK <version>\r\n ```

##### 4. delete

Delete a file. Returns FILE_NOT_FOUND error if file is not present.

Syntax:
```
	delete <file_name>\r\n
```

Response(If successful):


``` OK\r\n ```


### ERRORS
You may come across the following errors:
(Here is the meaning and the scenario in which they can occur)

``` ERR_CMD_ERR ```:                          Incorrect client command

``` ERR_FILE_NOT_FOUND ```:                   File requested is not found (Either expired or not created) 

``` ERR_VERSION <latest-version> ```:         Error in cas (Version mismatch)

``` ERR_INTERNAL ```:                         Internal server error 

``` ERR_REDIRECT <leader-ip:leader-port> ```: Redirect the client to the leader node



### Limitations 

1. Sometime leader may change after accepting commands and before replicating on majority and the new elected leader might not have the command in its log. In this case, the clients is stuck forever.

2. If server cannot figure out the end of command such as when numbytes is not numeric or incorrect number of fields in the command or unknown command etc, the client connection is closed.



### Test cases

A test file is written to test the server. Following scenarios are tested :

1. When servers are started, a leader is always elected. Clients attempting to do write, cas and delete at the follower are redirected to the leader. Updates committed at the leaders should get eventually applied to the followers file server.

2. Sending a sequence of commands including read, write, cas and delete to any of the server(not necessary leader). 

3. Writing binary content in the file.

4. File expiry time handling is tested. Any file that has a file expiry time is removed from the server once its time is expired 

5. N clients concurrently write to the same file. At the end, in the file should be any one clients' last write. 

6. N clients cas to the same file. At the end, in the file should be any one clients' last write.

7. What if a follower fails and come back later? Commands committed at the leader in between should be eventually reflected at follower's file server.

8. What if the leader dies and come back later ? New leader should be elected and when the previous leader come back, it should turn itself to follower state and all the commands committed at the new leader in between should be eventually reflected at its file server.

9. Killed 3 consecutively elected leaders and check if at the end all the changes committed by each of the leader are reflected in the file system.

10. What if minority of nodes are alive ? Client is stuck for this period and when the majority of nodes become alive it should get served.




### Contact

Prerna Gupta <br />
Email: prernagupta2592@gmail.com 
