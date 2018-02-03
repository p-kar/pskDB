package keyvalue

const (
    // Time interval between subsequent heartbeats in ms
    HEARTBEAT_TIME_INTERVAL = 10
    // Number of random servers that each server sends heartbeats to
    // in each gossip interval
    GOSSIP_HEARTBEAT_FANOUT = 2
    // Time interval in ms after which the failure detection algorithm starts
    GOSSIP_HEARTBEAT_TIMEOUT = 100
)