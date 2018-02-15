package main

import (
)

type ClientInfo struct {
    // unique id of the client
    Id string
    // ip address of the client
    IP_address string
    // rpc address of the client (IP_address + ":" + Port_num)
    Address string
    // port number of the client
    Port_num string
    // unique id of the server
    Server_Id string
    // server address
    Server_Address string
}
