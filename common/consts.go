package common

const (
    // Time interval master waits before pinging the server during startup
    SERVER_STARTUP_WAIT_TIME = 500
    // Number of times master tries to ping the server during startup
    CREATE_SERVER_NUM_RETRIES = 5
    // Time interval master waits before pinging the client during startup
    CLIENT_STARTUP_WAIT_TIME = 500
    // Number of times master tries to ping the server during startup
    CREATE_CLIENT_NUM_RETRIES = 5
    // Timeout for stabilize to complete
    SERVER_STABILIZE_TIMEOUT = 3000
)