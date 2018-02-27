package main

import (
    "net/rpc"
)

type ServerInfo struct {
    // unique id of the server
    Id string
    // server address
    Address string
    // RPC client object
    rpcClient *rpc.Client
}

// declares a NewServerInfo object in heap and returns pointer
func NewServerInfoHeap(id string, addr string) *ServerInfo {
    new_server_info := new(ServerInfo)

    new_server_info.Id = id
    new_server_info.Address = addr
    new_server_info.rpcClient = nil

    return new_server_info
}

type ClientInfo struct {
    // unique id of the client
    Id string
    // ip address of the client
    IP_address string
    // port number of the client
    Port_num string
    // rpc address of the client (IP_address + ":" + Port_num)
    Address string
    // map of servers this client is connected to (some might be dead)
    serv_map map[string]*ServerInfo
}