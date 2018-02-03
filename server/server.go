package main

import (
    // "fmt"
    "os"
    "net"
    "net/rpc"
    "time"
    "sync"
    "sort"
    "math/rand"
    "errors"
    "../logger"
)

// this server's info structure
// [TODO] if this not just only read only then will not be thread safe
// need to check when sequence number is updated
var currServerInfo ServerInfo
// global map that maintains info about all alive servers
// [TODO] need to make accesses to this thread safe (mutually exclusive)
var serverMap = make(map[string] *ServerInfo)
// mutex for serverMap
// [TODO] make this a RWMutex to allow concurrent reads
var mutex_server_map = &sync.Mutex{}
// Logging instance
var log *logger.Logger

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

type ServerListener int

// used to add a new server to the cluster
// [TODO] checking duplicate doesn't stop the new server from spawning
func (sl *ServerListener) JoinClusterAsServer(req *JoinClusterAsServerRequest, 
    reply *JoinClusterAsServerReply) error {
    // [TODO] check in the map for the client also because both sets of ids 
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // are in the same namespace
    // check if the server is already not present
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

    // [TODO] send new server notifications to all existing servers
    for _,serv_info := range serverMap {
        client := getRPCConnection(serv_info.Address)
        if client == nil {
            continue
        }
        new_server_not_req := NewServerNotificationRequest{
            Id: currServerInfo.Id,
            NewServerInfo: *new_server_info,
        }
        var new_server_not_reply Nothing
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
    req *NewServerNotificationRequest, reply *Nothing) error {
    log.Info.Printf("New server [ID: %s, Port Number: %s] notification received from server [ID: %s].\n", req.NewServerInfo.Id, 
        req.NewServerInfo.Port_num, req.Id)
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // if already in the server map do nothing
    if _, ok := serverMap[req.NewServerInfo.Id]; ok {
        *reply = false
        return nil
    }
    // otherwise add to the serverMap
    serverMap[req.NewServerInfo.Id] = NewServerInfoHeap(req.NewServerInfo)
    *reply = false
    return nil
}


func (sl *ServerListener) HeartbeatNotification(
    req *HeartbeatNotificationRequest, reply *Nothing) error {
    *reply = false
    // acquire lock on server map
    mutex_server_map.Lock()
    defer mutex_server_map.Unlock()
    // iterate over ServerInfoList and update sequence numbers
    for _, serv_info := range req.ServerInfoList {
        sid := serv_info.Id
        seq_num := serv_info.Heartbeat_seqnum
        // skip if sid matches current server id
        if sid == currServerInfo.Id {
            continue
        }
        // if new server is found in heartbeat notification add to list
        // make sure the new server isn't under suspicion
        if _, ok := serverMap[sid]; ok == false && serv_info.Suspicion == false{
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

func startHeartbeats() {
    counter := 0
    for {
        // increment heartbeat sequence number
        currServerInfo.Heartbeat_seqnum++
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
        hearbeat_req := HeartbeatNotificationRequest{
            Id:currServerInfo.Id,
        }
        var hearbeat_reply Nothing
        hearbeat_req.ServerInfoList = append(hearbeat_req.ServerInfoList,
            currServerInfo)
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
        // if counter % 5 == 0 {
        //     fmt.Printf("\n\n -------------- SERVER %s --------------\n", currServerInfo.Id)
        //     for _,serv_info := range serverMap {
        //         if serv_info.Alive {
        //             fmt.Printf("%s\t%s\t%d\t[  ALIVE  ]\n", serv_info.Id, serv_info.Address, serv_info.Heartbeat_seqnum)
        //         } else if serv_info.Suspicion {
        //             fmt.Printf("%s\t%s\t%d\t[ SUSPECT ]\n", serv_info.Id, serv_info.Address, serv_info.Heartbeat_seqnum)
        //         }
        //     }
        // }
        mutex_server_map.Unlock()
        time.Sleep(HEARTBEAT_TIME_INTERVAL * time.Millisecond)
        counter++
    }
}

func main() {
    if len(os.Args) < 2 {
        log.Panic.Panicln("New server id and port not provided.")
    }
    // init the log
    log = logger.NewLogger("[ SERVER " + os.Args[1] + " ] ", os.Stdout,
        os.Stdout, os.Stdout, os.Stderr, os.Stderr)

    currServerInfo = ServerInfo{
        Id: os.Args[1],
        IP_address: "localhost",
        Port_num: os.Args[2],
        Address: "localhost:" + os.Args[2],
        Heartbeat_seqnum: 1,
        Timestamp: time.Now(),
        Alive: true,
        Suspicion: false,
    }

    // connect to the cluster
    if len(os.Args) > 3 {
        for i := 3; i < len(os.Args); i ++ {
            client := getRPCConnection("localhost:" + os.Args[i])
            if client == nil {
                continue
            }
            join_cluster_req := JoinClusterAsServerRequest{
                Id: currServerInfo.Id, 
                IP_address: currServerInfo.IP_address, 
                Port_num: currServerInfo.Port_num,
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
        currServerInfo.IP_address + ":" + currServerInfo.Port_num)
    if err != nil {
        log.Panic.Panicf("Unable resolve TCP address %s.\n", 
            currServerInfo.IP_address + ":" + currServerInfo.Port_num)
    }

    inbound, err := net.ListenTCP("tcp", server_addr)
    if err != nil {
        log.Panic.Panicf("Unable listen at TCP address %s.\n", 
            currServerInfo.IP_address + ":" + currServerInfo.Port_num)
    }

    // start hearbeat thread
    go startHeartbeats()

    log.Info.Println("Server successfully started.")

    listener := new(ServerListener)
    rpc.Register(listener)
    rpc.Accept(inbound)
}