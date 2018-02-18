package main

const (
    // Time interval between subsequent heartbeats in ms
    HEARTBEAT_TIME_INTERVAL = 100
    // Number of random servers that each server sends heartbeats to
    // in each gossip interval
    GOSSIP_HEARTBEAT_FANOUT = 2
    // Time interval in ms after which the failure detection algorithm starts
    GOSSIP_HEARTBEAT_TIMEOUT = 1000
    // Stabilize round interval
    STABILIZE_ROUND_INTERVAL = 100
)