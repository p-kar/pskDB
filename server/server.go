package main

import (
    // "fmt"
    "os"
    "net"
    "math"
    "sort"
    "sync"
    "time"
    "errors"
    "strconv"
    "net/rpc"
    "math/rand"
    cc "../common"
)

// global map that maintains info about all alive servers
var serverMap = make(map[string]*ServerInfo)

// mutex for serverMap
// [TODO] make this a RWMutex to allow concurrent reads
var mutex_server_map = &sync.Mutex{}

// this server's info structure
var currServerInfo ServerInfo

// mutex for currServerInfo
var mutex_curr_server_info = &sync.Mutex{}

// Logging instance
var log *cc.Logger

// Kill Heartbeat
var doneHeartbeat = make(chan bool)

// List of blacklisted nodes
var blacklistMap = make(map[string]bool)

// mutex for blacklistMap
var mutex_blacklist_map = &sync.Mutex{}

// map for key value store
var keyValueMap = make(map[string] *KeyValueInfo)

// mutex for keyValueMap
var mutex_key_value_map = &sync.Mutex{}

// get RPC client object given an IP address - if not in the blacklist
func getRPCConnection(address string) *rpc.Client {

    if _, ok := blacklistMap[address]; ok == true {
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

// get Lamport timestamp from sequence number and id
func GetLamportTimestampFromSeqnum(seq_num int64, id string) float64 {
    nid, err := strconv.Atoi(id)
    if err != nil {
        log.Panic.Panicln("Non integer id number provided.")
    }
    num_id := float64(nid)
    num_id = num_id / math.Pow10(len(id))
    return float64(seq_num) + num_id
}

// get current Lamport timestamp
func GetLamportTimestamp() float64 {
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()
    return currServerInfo.Lamport_Timestamp
}

// Atomically get and increment Lamport's timestamp (needed for reads and writes)
func GetAndIncrementLamportTimestamp() float64 {
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()
    curr_lamport_time := currServerInfo.Lamport_Timestamp
    currServerInfo.Lamport_Timestamp += 1.0
    return curr_lamport_time
}

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
        return errors.New("JoinClusterAsServer: received duplicate server ids")
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
        client := getRPCConnection(serv_info.Address)
        if client == nil {
            continue
        }
        new_server_not_req := NewServerNotificationRequest{
            Id:            currServerInfo.Id,
            NewServerInfo: *new_server_info,
        }
        var new_server_not_reply cc.Nothing
        client.Go("ServerListener.NewServerNotification",
            &new_server_not_req, &new_server_not_reply, nil)
        client.Close()
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
    if currServerInfo.Id == req.NewServerInfoHeap.Id {
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
    mutex_blacklist_map.Lock()
    defer mutex_blacklist_map.Unlock()
    blacklistMap[req.Address] = true

    log.Info.Printf("Added server [Address: %s] to blacklist.\n", req.Address)

    return nil
}

func (sl *ServerListener) KillServerNotification(
    req *KillServerNotificationRequest, reply *cc.Nothing) error {
    nodeId := req.Id
    *reply = false
    // acquire lock on server map
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // delete nodeId from server map
    delete(serverMap, nodeId)
    log.Info.Printf("Received stop server notification from server [ID: %s].\n", nodeId)
    return nil
}

func (sl *ServerListener) KillServer(
    req *cc.Nothing, reply *cc.Nothing) error {
    *reply = false
    killserver_req := KillServerNotificationRequest{
        Id: currServerInfo.Id,
    }
    var killserver_reply cc.Nothing
    // send message to all servers that I got a kill request from the master
    for pid := range serverMap {
        client := getRPCConnection(serverMap[pid].Address)
        if client != nil {
            client.Go("ServerListener.KillServerNotification",
                &killserver_req, &killserver_reply, nil)
            client.Close()
        }
    }
    doneHeartbeat <- true
    return nil
}

// write a <K,V> pair in the key value store
func (sl *ServerListener) PutKVServer (
    req *cc.PutKVServerRequest, reply *cc.Nothing) error {
    // acquire lock on key value store
    mutex_key_value_map.Lock()
    defer mutex_key_value_map.Unlock()
    if _, ok := keyValueMap[req.Key]; ok == false {
        keyValueMap[req.Key] = new(KeyValueInfo)
        keyValueMap[req.Key].Key = req.Key
        keyValueMap[req.Key].Value = req.Value
        keyValueMap[req.Key].Version = GetAndIncrementLamportTimestamp()
    } else {
        curr_lamport_time := GetAndIncrementLamportTimestamp()
        if keyValueMap[req.Key].Version < curr_lamport_time {
            keyValueMap[req.Key].Value = req.Value
            keyValueMap[req.Key].Version = curr_lamport_time
        }
    }
    return nil
}

// read a <K,V> pair given K from the key value store
func (sl *ServerListener) GetKVServer (
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
    }
    return nil
}

// print the current membership list
func (sl *ServerListener) PrintMembershipList(
    req *cc.Nothing, reply *cc.Nothing) error {
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    var member_list string
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

func startHeartbeats() {
    for {
        select {
        case <-doneHeartbeat:
            log.Info.Printf("Server [ID: %s] killing hearbeat thread\n", currServerInfo.Id)
            return
        default:
            // increment heartbeat sequence number
            // increment lamport timestamp for send event
            mutex_curr_server_info.Lock()
            currServerInfo.Heartbeat_seqnum++
            currServerInfo.Lamport_Timestamp += 1.0
            mutex_curr_server_info.Unlock()

            mutex_server_map.Lock()
            // check if timeout reached for any server in membership list
            for sid := range serverMap {
                elapsed := time.Since(serverMap[sid].Timestamp).Nanoseconds() / 1E6
                if elapsed > GOSSIP_HEARTBEAT_TIMEOUT {
                    if serverMap[sid].Alive == true {
                        log.Warning.Printf("Suspect a crashed server [ID: %s, Port_num: %s].\n",
                            sid, serverMap[sid].Port_num)
                        serverMap[sid].Alive = false
                        serverMap[sid].Suspicion = true
                        serverMap[sid].Timestamp = time.Now()
                        continue
                    } else if serverMap[sid].Suspicion == true {
                        log.Warning.Printf("Deleting crashed server [ID: %s, Port_num: %s].\n",
                            sid, serverMap[sid].Port_num)
                        delete(serverMap, sid)
                    }
                }
            }
            // send out heartbeats
            mutex_curr_server_info.Lock()
            hearbeat_req := HeartbeatNotificationRequest{
                Id: currServerInfo.Id,
            }
            var hearbeat_reply cc.Nothing
            hearbeat_req.ServerInfoList = append(hearbeat_req.ServerInfoList,
                currServerInfo)
            mutex_curr_server_info.Unlock()
            for _, serv_info := range serverMap {
                hearbeat_req.ServerInfoList = append(hearbeat_req.ServerInfoList,
                    *serv_info)
            }
            num_recipients := GOSSIP_HEARTBEAT_FANOUT
            if num_recipients > len(serverMap) {
                num_recipients = len(serverMap)
            }
            perm := rand.Perm(len(serverMap))[:num_recipients]
            sort.Ints(perm)
            // log.Info.Println(perm)
            pidx := 0
            kidx := 0
            for pid := range serverMap {
                if pidx >= GOSSIP_HEARTBEAT_FANOUT {
                    break
                }
                if perm[pidx] == kidx {
                    // send heartbeat
                    client := getRPCConnection(serverMap[pid].Address)

                    if client != nil {
                        client.Go("ServerListener.HeartbeatNotification",
                            &hearbeat_req, &hearbeat_reply, nil)
                        client.Close()
                    }
                    pidx++
                }
                kidx++
            }

            mutex_server_map.Unlock()
            time.Sleep(HEARTBEAT_TIME_INTERVAL * time.Millisecond)
        }
    }
}

func main() {
    if len(os.Args) < 2 {
        log.Panic.Panicln("New server id and port not provided.")
    }
    // init the log
    log = cc.NewLogger("[ SERVER "+os.Args[1]+" ] ", os.Stdout,
        os.Stdout, os.Stdout, os.Stderr, os.Stderr)

    currServerInfo = ServerInfo{
        Id:                   os.Args[1],
        IP_address:           "localhost",
        Port_num:             os.Args[2],
        Address:              "localhost:" + os.Args[2],
        Heartbeat_seqnum:     1,
        Timestamp:            time.Now(),
        Alive:                true,
        Suspicion:            false,
        Lamport_Timestamp:    GetLamportTimestampFromSeqnum(1, os.Args[1]),
    }

    // connect to the cluster
    if len(os.Args) > 3 {
        for i := 3; i < len(os.Args); i++ {
            client := getRPCConnection("localhost:" + os.Args[i])
            if client == nil {
                continue
            }
            join_cluster_req := JoinClusterAsServerRequest{
                Id:         currServerInfo.Id,
                IP_address: currServerInfo.IP_address,
                Port_num:   currServerInfo.Port_num,
            }
            var join_cluster_reply JoinClusterAsServerReply
            err := client.Call("ServerListener.JoinClusterAsServer",
                &join_cluster_req, &join_cluster_reply)
            if err != nil {
                log.Warning.Printf("RPC call to join server at port number: %s failed.\n", os.Args[i])
                continue
            }
            client.Close()
            mutex_server_map.Lock()
            for ii := 0; ii < len(join_cluster_reply.ServerInfoList); ii++ {
                sid := join_cluster_reply.ServerInfoList[ii].Id
                log.Info.Printf("Found a new server [ID: %s, Port_num: %s].\n",
                    sid, join_cluster_reply.ServerInfoList[ii].Port_num)
                serverMap[sid] = NewServerInfoHeap(join_cluster_reply.ServerInfoList[ii])
            }
            mutex_server_map.Unlock()
            break
        }
    }

    server_addr, err := net.ResolveTCPAddr("tcp",
        currServerInfo.IP_address+":"+currServerInfo.Port_num)
    if err != nil {
        log.Panic.Panicf("Unable resolve TCP address %s.\n",
            currServerInfo.IP_address+":"+currServerInfo.Port_num)
    }

    inbound, err := net.ListenTCP("tcp", server_addr)
    if err != nil {
        log.Panic.Panicf("Unable listen at TCP address %s.\n",
            currServerInfo.IP_address+":"+currServerInfo.Port_num)
    }

    // start hearbeat thread
    go startHeartbeats()

    log.Info.Println("Server successfully started.")

    listener := new(ServerListener)
    rpc.Register(listener)
    rpc.Accept(inbound)
}
