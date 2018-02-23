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
    Address string
}

type BreakConnectionRequest struct {
    // unique id of the server/client
    Id string
    // address of the server / client 
    // needed for client to connect to server
    Address string
}

type PutKVClientRequest struct {
    // key
    Key string
    // value to be stored in the key value pair
    Value string
}

type PutKVServerRequest struct {
    // key
    Key string
    // value to be stored in the key value pair
    Value string
    // version of the last write/read on the key by the client
    Version float64
}

type PutKVServerReply struct {
    // key
    Key string
    // value to be stored in the key value pair
    Value string
    // lamport timestamp for this succeded request
    Version float64
}

type GetKVClientRequest struct {
    // key
    Key string
}

type GetKVClientReply struct {
    // key
    Key string
    // value returned by server
    Value string
    // lamport timestamp for this value
    Version float64
}

type GetKVServerRequest struct {
    // key
    Key string
    // minimum version of the key required
    Version float64
}

type GetKVServerReply struct {
    // key
    Key string
    // value returned by server
    Value string
    // lamport timestamp for this value
    Version float64
}

type StabilizeRequest struct {
    // number of rounds to perform the stabilization
    Rounds int
}
