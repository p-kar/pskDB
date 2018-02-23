# pskDB
Toy key value store for CS 380D (Distributed Computing I)

### Introduction

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