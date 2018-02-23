package main

import (
    // "fmt"
    cc "../common"
    "errors"
    "math"
    "time"
)

type ServerListener int

// used to add a new server to the cluster
// [TODO] checking duplicate doesn't stop the new server from spawning
func (sl *ServerListener) JoinClusterAsServer(req *JoinClusterAsServerRequest,
    reply *JoinClusterAsServerReply) error {
    // [TODO] check in the map for the client also because both sets of ids
    // are in the same namespace check if the server is already not present
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // acquire lock on currServerInfo
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()

    if _, ok := serverMap[req.Id]; ok {
        return nil
    }
    if currServerInfo.Id == req.Id {
        return errors.New("JoinClusterAsServer: received duplicate server ids")
    }
    log.Info.Printf("Server [ID: %s] received request to connect from server [ID: %s] at port number %s.\n",
        currServerInfo.Id, req.Id, req.Port_num)
    reply.ServerInfoList = append(reply.ServerInfoList, currServerInfo)
    for _, serv_info := range serverMap {
        reply.ServerInfoList = append(reply.ServerInfoList, *serv_info)
    }
    new_server_info := new(ServerInfo)

    new_server_info.Id = req.Id
    new_server_info.IP_address = req.IP_address
    new_server_info.Port_num = req.Port_num
    new_server_info.Address = req.IP_address + ":" + req.Port_num
    new_server_info.Heartbeat_seqnum = 1
    new_server_info.Timestamp = time.Now()
    new_server_info.Alive = true
    new_server_info.Suspicion = false

    for _, serv_info := range serverMap {
        if serv_info.rpcClient == nil {
            serv_info.rpcClient = getRPCConnection(serv_info.Address)
        }
        rpc_client := serv_info.rpcClient
        if rpc_client == nil {
            continue
        }
        new_server_not_req := NewServerNotificationRequest{
            Id:            currServerInfo.Id,
            NewServerInfo: *new_server_info,
        }
        var new_server_not_reply cc.Nothing
        rpc_call := rpc_client.Go("ServerListener.NewServerNotification",
            &new_server_not_req, &new_server_not_reply, nil)
        go AsyncCallHandler(rpc_call, serv_info.Id)
    }
    // adding the new server to the server map
    serverMap[req.Id] = new_server_info
    return nil
}

// used to notify servers in the cluster that a new server has been added
// this func is intended to be used as a non-blocking RPC call
func (sl *ServerListener) NewServerNotification(
    req *NewServerNotificationRequest, reply *cc.Nothing) error {
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // if already in the server map do nothing
    if _, ok := serverMap[req.NewServerInfo.Id]; ok {
        *reply = false
        return nil
    }
    // if the same as current server do nothing
    if currServerInfo.Id == req.NewServerInfo.Id {
        *reply = false
        return nil
    }
    // otherwise add to the serverMap
    log.Info.Printf("New server [ID: %s, Port Number: %s] notification received from server [ID: %s].\n", req.NewServerInfo.Id,
        req.NewServerInfo.Port_num, req.Id)
    serverMap[req.NewServerInfo.Id] = NewServerInfoHeap(req.NewServerInfo)
    *reply = false
    return nil
}

// used to get the server's ServerInfo structure
func (sl *ServerListener) GetServerInfo(
    req *cc.Nothing, reply *GetServerInfoReply) error {
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()
    reply.Id = currServerInfo.Id
    // copy structure contents
    reply.Info.Id = currServerInfo.Id
    reply.Info.IP_address = currServerInfo.IP_address
    reply.Info.Address = currServerInfo.Address
    reply.Info.Port_num = currServerInfo.Port_num
    reply.Info.Heartbeat_seqnum = currServerInfo.Heartbeat_seqnum
    reply.Info.Timestamp = time.Now()
    reply.Info.Alive = currServerInfo.Alive
    reply.Info.Suspicion = currServerInfo.Suspicion
    reply.Info.Lamport_Timestamp = currServerInfo.Lamport_Timestamp
    reply.Info.rpcClient = nil
    return nil
}

func (sl *ServerListener) HeartbeatNotification(
    req *HeartbeatNotificationRequest, reply *cc.Nothing) error {
    *reply = false
    // acquire lock on server map
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // acquire lock on currServerInfo
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()
    // iterate over ServerInfoList and update sequence numbers
    for _, serv_info := range req.ServerInfoList {
        sid := serv_info.Id
        seq_num := serv_info.Heartbeat_seqnum
        sender_lamport_time := GetLamportTimestampFromSeqnum(
            int64(math.Floor(serv_info.Lamport_Timestamp)),
            currServerInfo.Id,
        )
        // update Lamport's clock on current server
        currServerInfo.Lamport_Timestamp = math.Max(
            sender_lamport_time,
            currServerInfo.Lamport_Timestamp,
        ) + 1.0

        // skip if sid matches current server id
        if sid == currServerInfo.Id {
            continue
        }
        // if new server is found in heartbeat notification add to list
        // make sure the new server isn't under suspicion
        if _, ok := serverMap[sid]; ok == false && serv_info.Suspicion == false {
            log.Info.Printf("Found a new server [ID: %s, Port_num: %s].\n",
                sid, serv_info.Port_num)
            new_server_info := NewServerInfoHeap(serv_info)
            serverMap[sid] = new_server_info
            continue
        }
        // suspicion server don't add
        if _, ok := serverMap[sid]; ok == false {
            continue
        }
        // if stale sequence number skip
        if serverMap[sid].Heartbeat_seqnum >= seq_num {
            continue
        }
        // update sequence number and timestamp
        serverMap[sid].Heartbeat_seqnum = seq_num
        serverMap[sid].Timestamp = time.Now()
        // new sequence number seen remove from suspicion state
        if serverMap[sid].Suspicion == true {
            serverMap[sid].Suspicion = false
            serverMap[sid].Alive = true
        }
    }
    return nil
}

// Create connection between server/client and server. Remove node from blacklist to prevent server from making an RPC Connection
// [TODO] to restore connection, any sync reqd?
func (sl *ServerListener) CreateConnection(
    req *cc.CreateConnectionRequest, reply *cc.Nothing) error {
    // [TODO] do we need the id?
    var serv_info *ServerInfo
    mutex_curr_server_info.Lock()
    serv_info = NewServerInfoHeap(currServerInfo)
    mutex_curr_server_info.Unlock()
    // acquire lock blacklistMap lock
    mutex_blacklist_map.Lock()
    defer mutex_blacklist_map.Unlock()
    if _, ok := blacklistMap[req.Address]; ok == true {
        delete(blacklistMap, req.Address)
        log.Info.Printf("Removed server [ID: %s, Address: %s] from blacklist.\n", req.Id, req.Address)
    }
    rpc_client := getRPCConnection(req.Address)
    var new_server_not_reply cc.Nothing
    new_server_not_req := NewServerNotificationRequest{
        Id:            currServerInfo.Id,
        NewServerInfo: *serv_info,
    }
    if rpc_client != nil {
        err := rpc_client.Call("ServerListener.NewServerNotification",
            &new_server_not_req, &new_server_not_reply)
        if err != nil {
            log.Warning.Printf("Unable to reconnect to server [ID: %s, Address: %s]\n", req.Id, req.Address)
        }
        rpc_client.Close()
    }
    return nil
}

// Break connection between server/client and server. Add server/client to blacklist to prevent it from making an RPC Connection
func (sl *ServerListener) BreakConnection(
    req *cc.BreakConnectionRequest, reply *cc.Nothing) error {
    // [TODO] do we need the id?
    mutex_server_map.Lock()
    mutex_server_map.Unlock()
    mutex_blacklist_map.Lock()
    defer mutex_blacklist_map.Unlock()
    blacklistMap[req.Address] = true
    // need to close the connection
    if _, ok := serverMap[req.Id]; ok && serverMap[req.Id].rpcClient != nil{
        serverMap[req.Id].rpcClient.Close()
        serverMap[req.Id].rpcClient = nil
    }

    log.Info.Printf("Added server [Address: %s] to blacklist.\n", req.Address)

    return nil
}

func (sl *ServerListener) KillServerNotification(
    req *KillServerNotificationRequest, reply *cc.Nothing) error {
    *reply = false
    // acquire lock on server map
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // delete nodeId from server map
    if _, ok := serverMap[req.Id]; ok && serverMap[req.Id].rpcClient != nil{
        serverMap[req.Id].rpcClient.Close()
        serverMap[req.Id].rpcClient = nil
    }
    delete(serverMap, req.Id)
    log.Info.Printf("Received stop server notification from server [ID: %s].\n", req.Id)
    return nil
}

func (sl *ServerListener) KillServer(
    req *cc.Nothing, reply *cc.Nothing) error {
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    *reply = false
    killserver_req := KillServerNotificationRequest{
        Id: currServerInfo.Id,
    }
    var killserver_reply cc.Nothing
    // send message to all servers that I got a kill request from the master
    for _, serv_info := range serverMap {
        if serv_info.rpcClient == nil {
            serv_info.rpcClient = getRPCConnection(serv_info.Address)
        }
        rpc_client := serv_info.rpcClient
        if rpc_client != nil {
            rpc_client.Go("ServerListener.KillServerNotification",
                &killserver_req, &killserver_reply, nil)
            rpc_client.Close()
        }
    }
    doneHeartbeat <- true
    return nil
}

// write a <K,V> pair in the key value store
func (sl *ServerListener) PutKVServer (
    req *cc.PutKVServerRequest, reply *cc.PutKVServerReply) error {
    // acquire lock on key value store
    mutex_key_value_map.Lock()
    defer mutex_key_value_map.Unlock()
    if _, ok := keyValueMap[req.Key]; ok == false {
        keyValueMap[req.Key] = new(KeyValueInfo)
        keyValueMap[req.Key].Key = req.Key
        keyValueMap[req.Key].Value = req.Value
        if req.Version == -1 {   
            keyValueMap[req.Key].Version = GetAndIncrementLamportTimestamp()
        } else {
            keyValueMap[req.Key].Version = ConditionalIncrementLamportTimestamp(req.Version)
        }
        AddToWriteLog(keyValueMap[req.Key])
    } else {
        var curr_lamport_time float64
        if req.Version == -1 {
            curr_lamport_time = GetAndIncrementLamportTimestamp()
        } else {
            curr_lamport_time = ConditionalIncrementLamportTimestamp(req.Version)
            
        }
        if keyValueMap[req.Key].Version < curr_lamport_time {
            keyValueMap[req.Key].Value = req.Value
            keyValueMap[req.Key].Version = curr_lamport_time
            AddToWriteLog(keyValueMap[req.Key])
        }
    }
    reply.Key = keyValueMap[req.Key].Key
    reply.Value = keyValueMap[req.Key].Value
    reply.Version = keyValueMap[req.Key].Version
    return nil
}

// read a <K,V> pair given K from the key value store
func (sl *ServerListener) GetKVServer(
    req *cc.GetKVServerRequest, reply *cc.GetKVServerReply) error {
    // acquire lock on key value store
    mutex_key_value_map.Lock()
    defer mutex_key_value_map.Unlock()
    if val, ok := keyValueMap[req.Key]; ok == false {
        reply.Key = req.Key
        reply.Value = "ERR_KEY"
        reply.Version = GetAndIncrementLamportTimestamp()
    } else {
        GetAndIncrementLamportTimestamp()
        reply.Key = val.Key
        reply.Value = val.Value
        reply.Version = val.Version
        if val.Version < req.Version {
            reply.Value = "ERR_DEP"
        }
    }
    return nil
}

// print the current membership list
func (sl *ServerListener) PrintMembershipList(
    req *cc.Nothing, reply *cc.Nothing) error {
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // acquire lock on currServerInfo
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()
    member_list := "( *ID: " + currServerInfo.Id + ", Port: " + currServerInfo.Port_num + " ) "
    for id, serv_info := range serverMap {
        member_list = member_list + "( ID: " + id + ", Port: " + serv_info.Port_num + " ) "
    }
    log.Info.Printf("Member list: [ %s ].\n", member_list)
    return nil
}

// An RPC to check if server is alive
func (sl *ServerListener) PingServer(
    req *cc.Nothing, reply *cc.Nothing) error {
    return nil
}

// RPC call for master to stabilize the system
func (sl *ServerListener) Stabilize(
    req *cc.StabilizeRequest, reply *cc.StabilizeReply) error {

    log.Info.Println("Running stabilize....")
    reply.Id = currServerInfo.Id
    roundTicker := time.NewTicker(STABILIZE_ROUND_INTERVAL * time.Millisecond)
    defer roundTicker.Stop()
    
    // perform stabilize rounds    
    for i := 1; i<=req.Rounds; i++ {
        mutex_server_map.Lock()
        for _, serv_info := range serverMap {
            if serv_info.rpcClient == nil {
                serv_info.rpcClient = getRPCConnection(serv_info.Address)
            }
            rpc_client := serv_info.rpcClient
            if rpc_client == nil {
                continue
            }
            // send data to other server
            // assuming Id of the server doesn't change so not using Mutex
            send_key_data_req := SendKVDataRequest{
                                KeyValueMap : make(map[string]KeyValueInfo),
                                ServerId : currServerInfo.Id}
            mutex_key_value_map.Lock()
            for key, value := range keyValueMap {
                send_key_data_req.KeyValueMap[key] = *value
            }
            mutex_key_value_map.Unlock()

            var send_key_data_reply cc.Nothing
            err := rpc_client.Call("ServerListener.SendKVData", 
                &send_key_data_req, &send_key_data_reply)
            if err != nil {
                // [TODO] figure out what to do in this case
                if err == ErrShutdown {
                    serv_info.rpcClient = nil
                }
                continue
            }

        }
        mutex_server_map.Unlock()
        <-roundTicker.C
    }
    return nil
}

// RPC call to send KV store data to other servers
func (sl *ServerListener) SendKVData(
    req *SendKVDataRequest, reply *cc.Nothing) error {
    mutex_key_value_map.Lock()
    defer mutex_key_value_map.Unlock()
    // add/update keys from received data
    for key := range req.KeyValueMap {
        if _, ok := keyValueMap[key]; ok == false {
            keyValueMap[key] = NewKeyValueInfoHeap(req.KeyValueMap[key])
        } else {
            if req.KeyValueMap[key].Version > keyValueMap[key].Version {
                keyValueMap[key].Value = req.KeyValueMap[key].Value
                keyValueMap[key].Version = req.KeyValueMap[key].Version
            }
        }
    }
    return nil
}

// RPC call to print the whole KV store in this replica
func (sl *ServerListener) PrintStore(
    req *cc.Nothing, reply *cc.Nothing) error {
    mutex_curr_server_info.Lock()
    sid := currServerInfo.Id
    mutex_curr_server_info.Unlock()
    mutex_key_value_map.Lock()
    defer mutex_key_value_map.Unlock()
    // add/update keys from received data
    fmt.Printf("\nServer %s KV Store\n", sid)
    for key, value := range keyValueMap {
        fmt.Printf("%s\t:\t%s\n", key, value.Value)
    }
    fmt.Printf("\n")
    return nil
}
