# pskDB
Toy key value store for CS 380D (Distributed Computing I)

## Introduction

This project implements an eventually consistent key-value store. Each entry in the key-value store are a pair of strings. The system consists of clients and servers. Servers store the data and perform updates when asked by the clients. Clients are be able to perform the following operations:

* **Put**​ a new entry into the store
* **Get**​ the value associated with a key

The system guarantees eventual consistency plus two session guarantees:

* **Read Your Writes** - If a client has written a value to a key, it will never read an older value.
* **Monotonic Reads** - If a client has read a value, it will never read an older value.

If a client tries to perform a `get` operation when one of the properties isn’t satisfied, the replica that doesn’t satisfy the client’s dependencies returns `ERR_DEP`. If the key is not present instead, the replica will return `ERR_KEY`.

### Master Program API Specification

Master program provides a programmatic interface with the key-value store. The master program keeps track of and also sends command messages to all servers and
clients. More specifically, the master program reads a sequence of newline delineated commands from `stdin` ending with EOF which interact with the key-value store and, when instructed to, will display output from the key-value store to `stdout`.

|               Command              | Summary                                                                                                       |
|:----------------------------------:|---------------------------------------------------------------------------------------------------------------|
|          `joinServer [id]`         | Starts a server and will connect this server to all other servers in the system.                              |
|          `killServer [id]`         | Immediately kills a server. It blocks until the server is stopped.                                            |
| `joinClient [clientId] [serverId]` | Starts a client and connects it to the specified server.                                                      |
|    `breakConnection [id1] [id2]`   | Breaks the connection between a client and a server or between two servers.                                   |
|   `createConnection [id1] [id2]`   | Creates or restores the connection between a client and a server or between two servers.                      |
|             `stabilize`            | Blocks until all values are able to propagate to all connected servers.                                       |
|          `printStore [id]`         | Prints out a server’s key-value store.                                                                        |
|   `put [clientId] [key] [value]`   | Tells a client to associate the given value with the key.                                                     |
|       `get [clientId] [key]`       | Tells a client to attempt to get the key associated with the given value (can return `ERR_DEP` or `ERR_KEY`). |
|       `printMemberList [id]`       | Prints the server membership list of server [id].                                                             |

## Usage

### Requirements
* [Go 1.6.2](https://golang.org/)
* [Color](github.com/fatih/color) package for Go (The Makefile will automatically install it locally if you do not have it.)

### Compilation
The code can be compiled as follows:

~~~~
$ make
~~~~

### Running
All commands to the key-value store go through the `master`. Either the commands can be given to the master by `stdin` in console:

~~~~
$ ./master
~~~~

or you can pass a text file (`cmds.txt`) with the list of commands to be run:

~~~~
$ ./master < cmds.txt
~~~~

### Cleaning Up Stray Servers & Clients
It might happen that once you terminate the `master` program, a number of `serverNode` and `clientNode` process are up and later execution of `joinServer` and `joinClient` commands may throw errors because the port numbers might still be in use. The makefile allows you to automatically kill such processes.

~~~~
$ make cleanup
Killing stray server and client processes... done.
~~~~

## Design Details

### Membership Protocol

We employ a gossip-based membership protocol to detect failed servers and update the logical clocks for servers present in the cluster. Each server maintains a list of servers it believes to be a part of the cluster and has a heartbeat sequence number that monotonically increases. The server also maintains the latest heartbeat sequence number (`seq`) it has seen for each member in the cluster and records its local timestamp of when it received it. The protocol involves periodic pairwise inter-process interactions. In every round, each server randomly selects a few nodes (`GOSSIP_HEARTBEAT_FANOUT`) in its membership list, sends its entire membership list to those nodes and increments its own heartbeat sequence number. When a server receives a heartbeat notification it updates its own membership list replacing any outdated sequence numbers and add any new servers. If a server doesn't receive an updated heartbeat sequence number from a particular server within a given timeout period (`GOSSIP_HEARTBEAT_TIMEOUT`), that server is moved to a `SUSPICION` state and the timeout is reset. If this server sees a newer heartbeat sequence number within this timeout the server is moved back to `ALIVE` state, otherwise it is assumed to be dead and is removed from the membership list. We had to add the `SUSPICION` state inorder to avoid oscillations in the protocol. Note that we use local clocks at each server for timeouts to prevent clock skew or drift between servers to affect the protocol.

### Ensuring Consistency

The system provides eventual consistency along with two session guarantees - **Read Your Writes** and **Monotonic Reads**. In order to provide these two guarantees the client maintains additional state. For each key it keeps track of the latest version number it has seen for that key. During the first `put` or `get` request for that key the server replies back to the client with the version number of the key-value pair. The client records this value locally and all subsequent `put` or `get` calls to any server include this version number. During a `get` request the server checks this value with its own local version number, if it has a later value it returns that along with the new version number otherwise it returns `ERR_DEP`. If the server doesn't have that key in its own store it returns an `ERR_KEY` instead. After every `put` or `get` call the client updates the version number of the key appropriately. For providing eventual consistency each server sends out asynchronous `ProcessRemoteWrite` calls to other servers with the latest put requests.

### Stabilize Mechanism

The servers are made to reach the a consistent state by invoking the `stabilize` command. This is achieved by forwarding the entire key-value store data present on each server to every other server in its partition. The forwarding is done by using a synchronous flood message passing algorithm. When the `stabilize` command is invoked, the master passes this message to every server and the servers start the flood of their key-value store data in the network. The flooding algorithm runs for (n-1) rounds, where n is the number of servers in the system. Since all the servers are expected to run on a single system and with the assumption that size of the key-value store is small, we bound the time taken by each round to `STABILIZE_ROUND_INTERVAL`. In order to reduce the message size, we can have a log of the last exchanged info between the two servers in a way similar to Bayou. Also we are sending the entire key-value store just to simplify the implementation and this can also be optimized by having a log along with the asynchronous `put` requests we already have. At the end of all the rounds, each server replies to the master that stabilize is completed. When the master receives this from every server, it declares that the stabilize is completed.

### Test cases

We have tested for both correctness and performance of pskDB. 
## Correctness:
We ensure that the following cases are covered by the key-value store :
* Read-your-writes and monotonic reads session guarantees
* If client is reconnected to another, and the key is not present it should get ERR_KEY. If the latter server has an older value, it returns ERR_DEP
* Separating the servers into two partitions and checking if the guarantees in the previous point and eventual consistency, all hold in the induvidual partitions
* Ensure that stabilize operation occurs only within the partition
* After the partition is recovered, ensure that reads to a key after stabilize return the latest write at that server (Read-your-writes)

## Performance :

We build a system consisting of 5 servers and 25 clients. Then, we generate a random set of 2000 operations on the key-value store.

To run the performance test, execute the following command :

~~~~
$ time ./master < Tests/test_performance.txt > output_performance.txt
~~~~

For the 20000 requests, we get an average throughput of 588.23 requests/sec