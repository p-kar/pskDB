package main

// placeholder for RPC calls which don't reply
type Nothing bool

type BlackListInfo struct {
	// unique id of the server/client
	Id string
	// ip address of the node
	IP_address string
	// Port number of the node
	Port_num string
}
