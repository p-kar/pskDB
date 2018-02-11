package main

import (
	"os"
	"net"
	"sync"
	"net/rpc"
	cc "../common"
)

// Logging instance
var log *cc.Logger

// information about the client
var currClientInfo ClientInfo
// mutex for currClientInfo object
var mutex_client_info = &sync.Mutex{}

// get RPC client object given an IP address
func getRPCConnection(address string) *rpc.Client {
	if currClientInfo.Server_Id == "-1" {
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

// Create connection between client and server. Remove server from blacklist to prevent it from making an RPC Connection
// will replace any existing connection with some other server
func (cl *ClientListener) CreateConnection(
	req *cc.CreateConnectionRequest, reply *cc.Nothing) error {
	mutex_client_info.Lock()
	defer mutex_client_info.Unlock()
	currClientInfo.Server_Id = req.Id
	currClientInfo.Server_Address = req.Address
	log.Info.Printf("Client create connection to server [ID: %s, Address: %s].\n", req.Id, req.Address)
	return nil
}

// Break connection between client and server. Add server to blacklist to prevent it from making an RPC Connection
func (cl *ClientListener) BreakConnection(
	req *cc.BreakConnectionRequest, reply *cc.Nothing) error {
	mutex_client_info.Lock()
	defer mutex_client_info.Unlock()
	log.Info.Printf("---------> %s, %s\n", currClientInfo.Server_Id, req.Id)
	if currClientInfo.Server_Id == req.Id {
		log.Info.Printf("Client break connection to server [ID: %s, Address: %s].\n", req.Id, req.Address)
		currClientInfo.Server_Id = "-1"
	}
	return nil
}

// An RPC to check if client is alive
func (cl *ClientListener) PingClient(
	req *cc.Nothing, reply *cc.Nothing) error {
	return nil
}

func main() {
	if len(os.Args) < 5 {
		log.Panic.Panicln("New client id and port not provided.")
	}
	// init the log
	log = cc.NewLogger("[ CLIENT "+os.Args[1]+" ] ", os.Stdout,
		os.Stdout, os.Stdout, os.Stderr, os.Stderr)

	currClientInfo = ClientInfo{
		Id:         	os.Args[1],
		IP_address: 	"localhost",
		Port_num:   	os.Args[2],
		Address:    	"localhost:" + os.Args[2],
		Server_Id:		os.Args[3],
		Server_Address: os.Args[4],
	}

	client_addr, err := net.ResolveTCPAddr("tcp",
		currClientInfo.IP_address + ":" + currClientInfo.Port_num)
	if err != nil {
		log.Panic.Panicf("Unable resolve TCP address %s.\n",
			currClientInfo.IP_address + ":" + currClientInfo.Port_num)
	}

	inbound, err := net.ListenTCP("tcp", client_addr)
	if err != nil {
		log.Panic.Panicf("Unable listen at TCP address %s.\n",
			currClientInfo.IP_address + ":" + currClientInfo.Port_num)
	}

	log.Info.Printf("Client successfully started [ID: %s, Port_num: %s].\n", currClientInfo.Id, currClientInfo.Port_num)

	listener := new(ClientListener)
	rpc.Register(listener)
	rpc.Accept(inbound)
}
