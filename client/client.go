package main

import (
    "os"
    "net"
    "errors"
    "net/rpc"
    cc "../common"
)

// Logging instance
var log *cc.Logger

// information about the client
var clientInfo ClientInfo

// maintain state about previous reads/writes
var keyVersionInfo = make(map[string] float64)

// error message when rpc_client connection is shutdown
var ErrShutdown = errors.New("connection is shut down")

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

// Create connection between client and server. Remove server from blacklist to prevent it from making an RPC Connection
// will replace any existing connection with some other server
func (cl *ClientListener) CreateConnection(
    req *cc.CreateConnectionRequest, reply *cc.Nothing) error {
    if _, ok := clientInfo.serv_map[req.Id]; ok == false {
        log.Info.Printf("Client create connection to server [ID: %s, Address: %s].\n", req.Id, req.Address)
        clientInfo.serv_map[req.Id] = NewServerInfoHeap(req.Id, req.Address)
    }
    return nil
}

// Break connection between client and server.
// Remove server from the list of servers to prevent RPC connections to it
func (cl *ClientListener) BreakConnection(
    req *cc.BreakConnectionRequest, reply *cc.Nothing) error {
    if _, ok := clientInfo.serv_map[req.Id]; ok == true {
        log.Info.Printf("Client break connection to server [ID: %s, Address: %s].\n", req.Id, req.Address)
        if clientInfo.serv_map[req.Id].rpcClient != nil {
            clientInfo.serv_map[req.Id].rpcClient.Close()
        }
        clientInfo.serv_map[req.Id].rpcClient = nil
        delete(clientInfo.serv_map, req.Id)
    }
    return nil
}

// Asks client to execute a put request in the key value store 
// of the connected server.
func (cl *ClientListener) PutKVClient(
    req* cc.PutKVClientRequest, reply *cc.Nothing) error {
    for sid, _ := range clientInfo.serv_map {
        if clientInfo.serv_map[sid].rpcClient == nil {
            clientInfo.serv_map[sid].rpcClient = getRPCConnection(clientInfo.serv_map[sid].Address)
        }
        rpc_client := clientInfo.serv_map[sid].rpcClient
        if rpc_client != nil {
            var put_kv_server_req cc.PutKVServerRequest
            var put_kv_server_reply cc.PutKVServerReply

            put_kv_server_req.Key = req.Key
            put_kv_server_req.Value = req.Value
            if version, ok := keyVersionInfo[req.Key]; ok == true {
                put_kv_server_req.Version = version
            } else {
                put_kv_server_req.Version = -1 // doesnt care which version it receives
            }

            err := rpc_client.Call("ServerListener.PutKVServer",
                &put_kv_server_req, &put_kv_server_reply)
            if err == ErrShutdown {
                if clientInfo.serv_map[sid].rpcClient != nil {
                    clientInfo.serv_map[sid].rpcClient.Close()
                }
                clientInfo.serv_map[sid].rpcClient = nil
            } else if err != nil {
                continue
            }
            keyVersionInfo[put_kv_server_reply.Key] = put_kv_server_reply.Version
            return nil
        }
    }
    return errors.New("Unable to RPC dial server.")
}

// Asks client to execute a get request from the key value store 
// of the connected server.
func (cl *ClientListener) GetKVClient(
    req* cc.GetKVClientRequest, reply *cc.GetKVClientReply) error {
    for sid, _ := range clientInfo.serv_map {
        if clientInfo.serv_map[sid].rpcClient == nil {
            clientInfo.serv_map[sid].rpcClient = getRPCConnection(clientInfo.serv_map[sid].Address)
        }
        rpc_client := clientInfo.serv_map[sid].rpcClient
        if rpc_client != nil {
            var get_kv_server_req cc.GetKVServerRequest
            var get_kv_server_reply cc.GetKVServerReply

            get_kv_server_req.Key = req.Key
            if version, ok := keyVersionInfo[req.Key]; ok == true {
                get_kv_server_req.Version = version
            } else {
                get_kv_server_req.Version = -1 // doesnt care which version it receives
            }

            err := rpc_client.Call("ServerListener.GetKVServer",
                &get_kv_server_req, &get_kv_server_reply)
            if err == ErrShutdown {
                if clientInfo.serv_map[sid].rpcClient != nil {
                    clientInfo.serv_map[sid].rpcClient.Close()
                }
                clientInfo.serv_map[sid].rpcClient = nil
            } else if err != nil {
                continue
            }
            reply.Key = get_kv_server_reply.Key
            reply.Value = get_kv_server_reply.Value
            reply.Version = get_kv_server_reply.Version
            if get_kv_server_reply.Value != "ERR_DEP" && get_kv_server_reply.Value != "ERR_KEY" {
                keyVersionInfo[reply.Key] = get_kv_server_reply.Version
            }
            return nil
        }
    }
    return errors.New("Unable to RPC dial server.")
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

    // init the client object
    clientInfo = ClientInfo{
        Id:             os.Args[1],
        IP_address:     "localhost",
        Port_num:       os.Args[2],
        Address:        "localhost:" + os.Args[2],
    }
    clientInfo.serv_map = make(map[string]*ServerInfo)
    clientInfo.serv_map[os.Args[3]] = NewServerInfoHeap(os.Args[3], os.Args[4])

    client_addr, err := net.ResolveTCPAddr("tcp",
        clientInfo.IP_address + ":" + clientInfo.Port_num)
    if err != nil {
        log.Panic.Panicf("Unable resolve TCP address %s.\n",
            clientInfo.IP_address + ":" + clientInfo.Port_num)
    }

    inbound, err := net.ListenTCP("tcp", client_addr)
    if err != nil {
        log.Panic.Panicf("Unable listen at TCP address %s.\n",
            clientInfo.IP_address + ":" + clientInfo.Port_num)
    }

    log.Info.Printf("Client successfully started [ID: %s, Port_num: %s].\n", clientInfo.Id, clientInfo.Port_num)

    listener := new(ClientListener)
    rpc.Register(listener)
    rpc.Accept(inbound)
}
