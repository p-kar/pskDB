package main

import (
	"time"
)

// placeholder for RPC calls which don't reply
type Nothing bool

type ClientInfo struct {
	// unique id of the client
	Id string
	// ip address of the client
	IP_address string
	// rpc address of the client (IP_address + ":" + Port_num)
	Address string
	// port number of the client
	Port_num string
}

type ServerInfo struct {
	// unique id of the server
	Id string
	// ip address of the server
	IP_address string
	// rpc address of the server (IP_address + ":" + Port_num)
	Address string
	// port number of the server
	Port_num string
	// latest heartbeat sequence number that has been received from this server
	Heartbeat_seqnum int64
	// timestamp according to the local clock when the last sequence number
	// from this server was received
	Timestamp time.Time
	// boolean flag that is true when the server is alive and false if the
	// server is either in suspicion state or is dead
	Alive bool
	// boolean flag that is true when the server is in suspicion state
	Suspicion bool
}

func NewServerInfoHeap(serv_info ServerInfo) *ServerInfo {
	new_server_info := new(ServerInfo)

	new_server_info.Id = serv_info.Id
	new_server_info.IP_address = serv_info.IP_address
	new_server_info.Address = serv_info.Address
	new_server_info.Port_num = serv_info.Port_num

	return new_server_info
}

type JoinServerRequest struct {
	// unique identifier of the server that client wants to join
	Id string
	// ip address of the server that client wants to join
	IP_address string
	// port number for the server that client wants to join
	Port_num string
}

// the reply returns the server info connected to by the client
type JoinServerReply struct {
	// list of server already in the cluster
	CurrServerInfo *ServerInfo
}

type BlackListInfo struct {
	// unique id of the server/client
	Id string
	// ip address of the node
	IP_address string
	// Port number of the node
	Port_num string
}
