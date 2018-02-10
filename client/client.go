package main

import (
	// "fmt"
	"../logger"
	"net"
	"net/rpc"
	"os"
)

// Logging instance
var log *logger.Logger

// server info map
var serverMap = make(map[string]*ServerInfo)

// information about the client
var currClientInfo ClientInfo

// get RPC client object given an IP address
func getRPCConnection(address string) *rpc.Client {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		// log.Warning.Printf("Unable to dial server at address: %s.\n",
		//     address)
		return nil
	}
	return client
}

type ClientListener int

func (sl *ClientListener) Ping(
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

			join_server_req := JoinServerRequest{
				Id:         currClientInfo.Id,
				IP_address: currClientInfo.IP_address,
				Port_num:   currClientInfo.Port_num,
			}
			var join_server_reply JoinServerReply
			err := server.Call("ServerListener.JoinClientToServer",
				&join_server_req, &join_server_reply)

			if err == nil {
				server_id := join_server_reply.CurrServerInfo.Id
				log.Info.Printf("Found the server [ID: %s, Port_num: %s].\n",
					server_id, join_server_reply.CurrServerInfo.Port_num)
				serverMap[server_id] = NewServerInfoHeap(*join_server_reply.CurrServerInfo)
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
