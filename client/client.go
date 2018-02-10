package main

import (
	// "fmt"
	"../logger"
	"net"
	"net/rpc"
	"os"
	"sync"
)

// Logging instance
var log *logger.Logger

// server info
var serverInfo *ServerInfo

var mutex_server_map = &sync.Mutex{}

// information about the client
var currClientInfo ClientInfo

// blacklisted server list
var blacklistServerMap = make(map[string]bool)

var mutex_blacklistServer_map = &sync.Mutex{}

// get RPC client object given an IP address
func getRPCConnection(address string) *rpc.Client {
	if _, ok := blacklistServerMap[address]; ok == true {
		return nil
	}
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		// log.Warning.Printf("Unable to dial server at address: %s.\n",
		//     address)
		return nil
	}
	return client
}

type ClientListener int

// func (cl *ClientListener) GetClientInfo(
// 	req *Nothing, reply *ClientInfo) error {
// 	reply.Address = currClientInfo.Address
// 	reply.IP_address = currClientInfo.IP_address
// 	reply.Id = currClientInfo.Id
// 	reply.Port_num = currClientInfo.Port_num

// 	return nil
// }

// Create connection between client and server. Remove server from blacklist to prevent it from making an RPC Connection
// [TODO] to restore connection, any sync reqd?
func (cl *ClientListener) CreateConnection(
	req *BlackListInfo, reply *Nothing) error {

	nodeId := req.Id
	Address := req.IP_address + ":" + req.Port_num
	*reply = false

	if _, ok := blacklistServerMap[Address]; ok == true {
		// mutex_blacklistServer_map.Lock()
		delete(blacklistServerMap, Address)
		// mutex_blacklistServer_map.Unlock()
		log.Info.Printf("Removed Server [ID: %s, Port_num: %s] from blacklist.\n", nodeId, req.Port_num)
	}
	if nodeId != serverInfo.Id {
		server := getRPCConnection(Address)
		if server != nil {

			var join_server_req bool = true
			var join_server_reply *JoinServerReply
			// Get server info via RPC call
			err := server.Call("ServerListener.JoinClientToServer",
				&join_server_req, &join_server_reply)

			if err == nil && join_server_reply != nil {
				server_id := join_server_reply.CurrServerInfo.Id
				log.Info.Printf("Found the server [ID: %s, Port_num: %s].\n",
					server_id, join_server_reply.CurrServerInfo.Port_num)
				mutex_server_map.Lock()
				serverInfo = NewServerInfoHeap(*join_server_reply.CurrServerInfo)
				mutex_server_map.Unlock()
			} else {
				log.Warning.Printf("RPC call to join server at port number: %s failed.\n", os.Args[3])

			}
			server.Close()
		}
	}
	return nil
}

// Break connection between client and server. Add server to blacklist to prevent it from making an RPC Connection
func (cl *ClientListener) BreakConnection(
	req *BlackListInfo, reply *Nothing) error {
	// nodeId := req.Id
	Address := req.IP_address + ":" + req.Port_num
	// acquire lock for blacklistServerMap
	mutex_blacklistServer_map.Lock()
	blacklistServerMap[Address] = true
	mutex_blacklistServer_map.Unlock()

	log.Info.Printf("Added Server [Address: %s] to blacklist.\n", Address)

	return nil
}

// An RPC to check if client is alive
func (cl *ClientListener) Ping(
	req *Nothing, reply *Nothing) error {
	*reply = *req
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Panic.Panicln("New client id and port not provided.")
	}
	// init the log
	log = logger.NewLogger("[ CLIENT "+os.Args[1]+" ] ", os.Stdout,
		os.Stdout, os.Stdout, os.Stderr, os.Stderr)

	currClientInfo = ClientInfo{
		Id:         os.Args[1],
		IP_address: "localhost",
		Port_num:   os.Args[2],
		Address:    "localhost:" + os.Args[2],
	}

	// connect to the server
	if len(os.Args) > 3 {

		server := getRPCConnection("localhost:" + os.Args[3])
		if server != nil {

			var join_server_req bool = true
			var join_server_reply *JoinServerReply
			// Get server info via RPC call
			err := server.Call("ServerListener.JoinClientToServer",
				&join_server_req, &join_server_reply)

			if err == nil && join_server_reply != nil {
				server_id := join_server_reply.CurrServerInfo.Id
				log.Info.Printf("Found the server [ID: %s, Port_num: %s].\n",
					server_id, join_server_reply.CurrServerInfo.Port_num)
				mutex_server_map.Lock()
				serverInfo = NewServerInfoHeap(*join_server_reply.CurrServerInfo)
				mutex_server_map.Unlock()
			} else {
				log.Warning.Printf("RPC call to join server at port number: %s failed.\n", os.Args[3])

			}
			server.Close()
		}
	}

	server_addr, err := net.ResolveTCPAddr("tcp",
		currClientInfo.IP_address+":"+currClientInfo.Port_num)
	if err != nil {
		log.Panic.Panicf("Unable resolve TCP address %s.\n",
			currClientInfo.IP_address+":"+currClientInfo.Port_num)
	}

	inbound, err := net.ListenTCP("tcp", server_addr)
	if err != nil {
		log.Panic.Panicf("Unable listen at TCP address %s.\n",
			currClientInfo.IP_address+":"+currClientInfo.Port_num)
	}

	log.Info.Printf("Client successfully started [ID: %s, Port_num: %s].\n", currClientInfo.Id, currClientInfo.Port_num)

	listener := new(ClientListener)
	rpc.Register(listener)
	rpc.Accept(inbound)
	// go rpc.Wait()
}
