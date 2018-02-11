package common

import (
	// "time"
)

// placeholder for RPC calls which don't reply
type Nothing bool

type CreateConnectionRequest struct {
	// unique id of the server/client
	Id string
	// address of the server / client 
	// needed for client to connect to server
	Address string
}

type BreakConnectionRequest struct {
	// unique id of the server/client
	Id string
	// address of the server / client 
	// needed for client to connect to server
	Address string
}

type PutKeyValueRequest struct {
	// key
	Key string
	// value to be stored in the key value pair
	Value string
}
