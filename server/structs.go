package main

import (
    "time"
)

type ServerInfo struct {
    // unique id of the server
    Id string
    // ip address of the server
    IP_address string
    // rpc address of the server (IP_address + ":" + Port_num)
    Address string
    // port number of the server
    Port_num string
    // latest heartbeat sequence number that has been received from this server
    Heartbeat_seqnum int64
    // timestamp according to the local clock when the last sequence number
    // from this server was received
    Timestamp time.Time
    // boolean flag that is true when the server is alive and false if the
    // server is either in suspicion state or is dead
    Alive bool
    // boolean flag that is true when the server is in suspicion state
    Suspicion bool
    // Lamport's logical timestamp [ time.pid ]
    Lamport_Timestamp float64
}

// declares a NewServerInfo object in heap and returns pointer
func NewServerInfoHeap(serv_info ServerInfo) *ServerInfo {
    new_server_info := new(ServerInfo)

    new_server_info.Id = serv_info.Id
    new_server_info.IP_address = serv_info.IP_address
    new_server_info.Port_num = serv_info.Port_num
    new_server_info.Address = serv_info.Address
    new_server_info.Heartbeat_seqnum = serv_info.Heartbeat_seqnum
    new_server_info.Timestamp = time.Now()
    new_server_info.Alive = serv_info.Alive
    new_server_info.Suspicion = serv_info.Suspicion
    new_server_info.Lamport_Timestamp = serv_info.Lamport_Timestamp

    return new_server_info
}

type JoinClusterAsServerRequest struct {
    // unique identifier of the server that wants to join
    Id string
    // ip address of the server that wants to join
    IP_address string
    // port number for the server that wants to join
    Port_num string
}

// the reply returns a list of servers that are connected in the cluster
type JoinClusterAsServerReply struct {
    // list of server already in the cluster
    ServerInfoList []ServerInfo
}

// the reply returns the ServerInfo structure for replying server
type GetServerInfoReply struct {
    // Id of the replying server
    Id string
    // ServerInfo structure of the replying server
    Info ServerInfo
}

type NewServerNotificationRequest struct {
    // unique identifier of the server that sends the notification
    Id string
    // new server info
    NewServerInfo ServerInfo
}

type HeartbeatNotificationRequest struct {
    // unique identifier of the server that sends the notification
    Id string
    // list of server already in the cluster
    ServerInfoList []ServerInfo
}

type KillServerNotificationRequest struct {
    // unique id of the server that received kill request from master
    Id string
}

type KeyValueInfo struct {
    // K in the <K,V> pair
    Key string
    // V in the <K,V> pair
    Value string
    // lamport timestamp when this <K,V> pair was updated
    Version float64
}

type SendKVDataRequest struct {
    // entire Key value data
    KeyValueMap map[string]KeyValueInfo
    // source server Id
    ServerId string
}

// declares a new KeyValueInfo object in heap and returns a pointer
func NewKeyValueInfoHeap (info KeyValueInfo) *KeyValueInfo {
    new_key_value := new(KeyValueInfo)

    new_key_value.Key = info.Key
    new_key_value.Value = info.Value
    new_key_value.Version = info.Version

    return new_key_value
}
