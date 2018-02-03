package keyvalue

import (
	"time"
)

// placeholder for RPC calls which don't reply
type Nothing bool

type ServerInfo struct {
    // unique id of the server
    Id string
    // ip address of the server
    IP_address string
    // port number of the server
    Port_num string
    // latest heartbeat sequence number that has been received from this server
    Heartbeat_seqnum int
    // timestamp according to the local clock when the last sequence number
    // from this server was received
    Timestamp time.Time
    // boolean flag that is true when the server is alive and false if the 
    // server is either in suspicion state or is dead
    Alive bool
    // boolean flag that is true when the server is in suspicion state
    Suspicion bool
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

type NewServerNotificationRequest struct {
	// unique identifier of the server that sends the notification
    Id string
	// new server info
	NewServerInfo ServerInfo
}