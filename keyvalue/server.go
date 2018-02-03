package keyvalue

import (
    "os"
    "fmt"
    "net"
    "net/rpc"
    "time"
    "errors"
    "../logger"
)

// this server's info structure
var currServerInfo ServerInfo
// global map that maintains info about all alive servers
var serverMap = make(map[string] ServerInfo)
// Logging instance
var log = logger.NewLogger(os.Stdout, os.Stdout, 
        os.Stdout, os.Stderr, os.Stderr)

type ServerListener int

// used to add a new server to the cluster
func (sl *ServerListener) JoinClusterAsServer(req *JoinClusterAsServerRequest, 
    reply *JoinClusterAsServerReply) error {
    // [TODO] check in the map for the client also because both sets of ids 
    // are in the same namespace
    // check if the server is already not present
    if _, ok := serverMap[req.Id]; ok {
        return errors.New("JoinClusterAsServer: received duplicate server ids")
    }
    log.Info.Printf(`Server [ID: %s] received request 
        to connect from server [ID: %s] at port number %s.\n`, 
        currServerInfo.Id, req.Id, req.Port_num)
    reply.ServerInfoList = append(reply.ServerInfoList, currServerInfo)
    for _, serv_info := range serverMap {
        reply.ServerInfoList = append(reply.ServerInfoList, serv_info)
    }
    // adding the new server to the server map
    serverMap[req.Id] = ServerInfo{Id: req.Id, IP_address: req.IP_address, 
        Port_num: req.Port_num, Heartbeat_seqnum: 1, Timestamp: time.Now(), 
        Alive: true, Suspicion: false}
    // [TODO] send new server notifications to all existing servers
    return nil
}

// used to notify servers in the cluster that a new server has been added
// this func is intended to be used as a non-blocking RPC call
func (sl *ServerListener) NewServerNotification(
    req *NewServerNotificationRequest, reply *Nothing) error {
    log.Info.Printf(`New server [ID: %s, Port Number: %s] notification 
        received from server [ID: %s].\n`, req.NewServerInfo.Id, 
        req.NewServerInfo.Port_num, req.Id)
    // if already in the server map do nothing
    if _, ok := serverMap[req.NewServerInfo.Id]; ok {
        *reply = false
        return nil
    }
    serverMap[req.NewServerInfo.Id] = req.NewServerInfo
    *reply = false
    return nil
}

func main() {
    fmt.Println(os.Args)
    if len(os.Args) < 2 {
        log.Panic.Panicln("New server id and port not provided.")
    }

    currServerInfo = ServerInfo{Id: os.Args[1], IP_address: "localhost", 
    Port_num: os.Args[2], Heartbeat_seqnum: 1, Timestamp: time.Now(), 
    Alive: true, Suspicion: false}

    // connect to the cluster
    if len(os.Args) > 3 {
        for i := 3; i < len(os.Args); i ++ {
            client, err := rpc.Dial("tcp", "localhost:" + os.Args[i])
            if err != nil {
                log.Warning.Printf("Unable to dial server at port number: %s.\n", 
                    os.Args[i])
                continue
            }
            join_cluster_req := JoinClusterAsServerRequest{
                Id: currServerInfo.Id, 
                IP_address: currServerInfo.IP_address, 
                Port_num: currServerInfo.Port_num,
            }
            var join_cluster_reply JoinClusterAsServerReply
            err = client.Call("ServerListener.JoinClusterAsServer", 
                &join_cluster_req, &join_cluster_reply)
            if err != nil {
                log.Warning.Printf(`RPC call to join server at port number: %s 
                    failed.\n`, os.Args[i])
                continue
            }
            for ii := 0; ii < len(join_cluster_reply.ServerInfoList); ii++ {
                sid := join_cluster_reply.ServerInfoList[ii].Id
                log.Info.Printf("Found a new server [ID: %s, Port_num: %s].\n",
                    sid, join_cluster_reply.ServerInfoList[ii].Port_num)
                serverMap[sid] = join_cluster_reply.ServerInfoList[ii]
            }
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

    listener := new(ServerListener)
    rpc.Register(listener)
    rpc.Accept(inbound)
}