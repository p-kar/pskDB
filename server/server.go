package main

import (
    cc "../common"
    "math"
    "math/rand"
    "net"
    "net/rpc"
    "os"
    "sort"
    "strconv"
    "sync"
    "time"
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
var keyValueMap = make(map[string]*KeyValueInfo)

// mutex for keyValueMap
var mutex_key_value_map = &sync.Mutex{}

// write log for updates
var writeLog []*KeyValueInfo

// mutex for writeLog
var mutex_write_log = &sync.Mutex{}

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

// append write to the writeLog, creates a deep copy of the info
func AddToWriteLog(info *KeyValueInfo) {
    mutex_write_log.Lock()
    defer mutex_write_log.Unlock()
    new_update := NewKeyValueInfoHeap(*info)
    writeLog = append(writeLog, new_update)
}

// Atomically get and increment Lamport's timestamp (needed for reads and writes)
func GetAndIncrementLamportTimestamp() float64 {
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()
    curr_lamport_time := currServerInfo.Lamport_Timestamp
    currServerInfo.Lamport_Timestamp += 1.0
    return curr_lamport_time
}

// Atomically increment the lamport timestamp based on a pivot 
// (use only when put request with later timestamp is received)
func ConditionalIncrementLamportTimestamp(timestamp_pivot float64) float64 {
    mutex_curr_server_info.Lock()
    defer mutex_curr_server_info.Unlock()
    difference := currServerInfo.Lamport_Timestamp + 1.0 - timestamp_pivot
    if difference > 0 {
        currServerInfo.Lamport_Timestamp += 1.0
    } else {
        currServerInfo.Lamport_Timestamp += (1.0 + math.Ceil(difference))
    }
    return currServerInfo.Lamport_Timestamp
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
        Id:                os.Args[1],
        IP_address:        "localhost",
        Port_num:          os.Args[2],
        Address:           "localhost:" + os.Args[2],
        Heartbeat_seqnum:  1,
        Timestamp:         time.Now(),
        Alive:             true,
        Suspicion:         false,
        Lamport_Timestamp: GetLamportTimestampFromSeqnum(1, os.Args[1]),
    }

    // connect to the cluster
    if len(os.Args) > 3 {
        for i := 3; i < len(os.Args); i++ {
            rpc_client := getRPCConnection("localhost:" + os.Args[i])
            if rpc_client == nil {
                continue
            }
            join_cluster_req := JoinClusterAsServerRequest{
                Id:         currServerInfo.Id,
                IP_address: currServerInfo.IP_address,
                Port_num:   currServerInfo.Port_num,
            }
            var join_cluster_reply JoinClusterAsServerReply
            err := rpc_client.Call("ServerListener.JoinClusterAsServer",
                &join_cluster_req, &join_cluster_reply)
            rpc_client.Close()
            if err != nil {
                log.Warning.Printf("RPC call to join server at port number: %s failed.\n", os.Args[i])
                continue
            }
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

    // connect to partitioned nodes
    mutex_server_map.Lock()
    for i := 3; i < len(os.Args); i++ {
        already_present := false
        // check if this port num is already present in the member list
        for _, serv_info := range serverMap {
            if serv_info.Port_num != os.Args[i] {
                continue
            }
            already_present = true
            break
        }
        if already_present == true {
            continue
        }
        var retries = 0
        for retries < CONNECT_SERVER_NUM_RETRIES {
            rpc_client := getRPCConnection("localhost:" + os.Args[i])
            if rpc_client != nil {
                var req cc.Nothing
                var reply GetServerInfoReply
                err := rpc_client.Call("ServerListener.GetServerInfo", &req, &reply)
                rpc_client.Close()
                if err == nil {
                    log.Info.Printf("Found a new server [ID: %s, Port_num: %s].\n",
                        reply.Id, reply.Info.Port_num)
                    serverMap[reply.Id] = NewServerInfoHeap(reply.Info)
                    break
                }
            }
            time.Sleep(CONNECT_SERVER_REQUEST_DELAY * time.Millisecond)
            retries += 1
        }
    }
    mutex_server_map.Unlock()

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
