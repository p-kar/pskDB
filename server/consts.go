package main

const (
	// Number of retries to connect to servers
	CONNECT_SERVER_NUM_RETRIES = 3
	// Time interval in ms before retrying connection to server
	CONNECT_SERVER_REQUEST_DELAY = 100
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