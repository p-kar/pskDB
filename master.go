package main

import (
	cc "./common"
	"bufio"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

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

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	log := cc.NewLogger("[  MASTER  ] ", os.Stdout, os.Stdout,
		os.Stdout, os.Stderr, os.Stderr)
	// map from node(server) id to its port number
	serverNodeMap := make(map[string]int)
	// Starting value of the port numbers used by servers
	serverNextPort := 9000
	// map from node(client) id to its port number
	clientNodeMap := make(map[string]int)
	// Starting value of the port numbers used by clients
	clientNextPort := 10000

	for scanner.Scan() {
		// read and process the command from stdin
		command := scanner.Text()
		commandSplit := strings.Split(command, " ")

		if commandSplit[0] == "done" {
			log.Info.Println("Goodbye.")
			break
		}

		switch commandSplit[0] {
		case "joinServer":
			log.Info.Println("Executing...", commandSplit)

			// Create startup info i.e., arguments
			// example arguments ["1" "9001" "9004" "9003"]
			nodeId := commandSplit[1]
			args := []string{}
			args = append(args, nodeId, strconv.Itoa(serverNextPort))
			for _, port := range serverNodeMap {
				args = append(args, strconv.Itoa(port))
			}
			serverNodeMap[nodeId] = serverNextPort
			serverNextPort++

			// spawn the server
			joinCmd := exec.Command("./serverNode", args...)
			joinCmd.Stdout = os.Stdout
			joinCmd.Stderr = os.Stderr
			err := joinCmd.Start()
			if err != nil {
				log.Panic.Panicln(err)
			}
			// log.Info.Println("Started server with process id", joinCmd.Process.Pid)
			go joinCmd.Wait()

			req := true
			var reply bool
			var calls int = 0
			for {
				calls = calls + 1
				rpc_client := getRPCConnection("localhost:" + strconv.Itoa(serverNodeMap[nodeId]))
				if rpc_client != nil {
					err := rpc_client.Call("ServerListener.PingServer", &req, &reply)
					if err != nil {
						rpc_client.Close()
						continue
					}
					break
				}
				if calls > cc.CREATE_SERVER_NUM_RETRIES {
					log.Warning.Printf("Killing server %d\n", joinCmd.Process.Pid)
					killCmd := exec.Command("kill", strconv.Itoa(joinCmd.Process.Pid))
					killCmd.Stdout = os.Stdout
					killCmd.Stderr = os.Stderr
					err := killCmd.Start()
					if err != nil {
						log.Panic.Panicln(err)
					}
					break
				}
				time.Sleep(cc.SERVER_STARTUP_WAIT_TIME * time.Millisecond)
			}

		case "killServer":
			log.Info.Println("Executing...", commandSplit)
			// get Node ID
			nodeId := commandSplit[1]
			// get server port number for RPC call
			if _, ok := serverNodeMap[nodeId]; ok == false {
				log.Warning.Printf("Server [ID: %s] is not present in the cluster\n", nodeId)
				continue
			}
			serverPort := serverNodeMap[nodeId]
			rpc_client := getRPCConnection("localhost:" + strconv.Itoa(serverPort))
			if rpc_client != nil {
				var req, reply cc.Nothing
				err := rpc_client.Call("ServerListener.KillServer", &req, &reply)
				if err != nil {
					log.Warning.Printf("Kill server [ID: %s] command failed.", nodeId)
				}
				// log.Info.Println("killServer finished")
				delete(serverNodeMap, nodeId)
				rpc_client.Close()
			} else {
				log.Warning.Println("getRPCConnection returned a nil value.")
			}

		case "joinClient":
			log.Info.Println("Executing...", commandSplit)
			// get server and client IDs
			clientNodeId := commandSplit[1]
			serverNodeId := commandSplit[2]
			if _, ok := serverNodeMap[clientNodeId]; ok == true {
				log.Warning.Printf("Server ID: %s already present in the cluster\n", clientNodeId)
				continue
			} else if _, ok := clientNodeMap[serverNodeId]; ok == true {
				log.Warning.Printf("Client ID: %s already present in the cluster\n", serverNodeId)
				continue
			} else if _, ok := clientNodeMap[clientNodeId]; ok == true {
				log.Warning.Printf("Client ID: %s already present in the cluster\n", clientNodeId)
				continue
			} else if _, ok := serverNodeMap[serverNodeId]; ok == false {
				log.Warning.Printf("Server ID: %s not present in the cluster\n", clientNodeId)
				continue
			}
			server_address := "localhost:" + strconv.Itoa(serverNodeMap[serverNodeId])
			args := []string{}
			args = append(
				args,
				clientNodeId,
				strconv.Itoa(clientNextPort),
				serverNodeId,
				server_address,
			)
			clientNodeMap[clientNodeId] = clientNextPort
			clientNextPort++

			// spawn the client
			joinCmd := exec.Command("./clientNode", args...)
			joinCmd.Stdout = os.Stdout
			joinCmd.Stderr = os.Stderr
			err := joinCmd.Start()
			if err != nil {
				log.Panic.Panicln(err)
			}
			// log.Info.Println("Started server with process id", joinCmd.Process.Pid)
			go joinCmd.Wait()

			req := true
			var reply bool
			var calls int = 0

			for {
				calls++
				rpc_client := getRPCConnection("localhost:" + strconv.Itoa(clientNodeMap[clientNodeId]))
				if rpc_client != nil {
					err := rpc_client.Call("ClientListener.PingClient", &req, &reply)
					if err != nil {
						rpc_client.Close()
						continue
					}
					break
				}
				if calls > cc.CREATE_CLIENT_NUM_RETRIES {
					log.Warning.Printf("Killing client %d\n", joinCmd.Process.Pid)
					killCmd := exec.Command("kill", strconv.Itoa(joinCmd.Process.Pid))
					killCmd.Stdout = os.Stdout
					killCmd.Stderr = os.Stderr
					err := killCmd.Start()
					if err != nil {
						log.Panic.Panicln(err)
					}
					break
				}
				time.Sleep(cc.CLIENT_STARTUP_WAIT_TIME * time.Millisecond)
			}

		case "breakConnection":
			log.Info.Println("Executing...", commandSplit)

			// get node IDs
			nodeId1 := commandSplit[1]
			nodeId2 := commandSplit[2]

			_, ok_1 := serverNodeMap[nodeId1]
			_, ok_2 := clientNodeMap[nodeId1]

			_, ok_3 := serverNodeMap[nodeId2]
			_, ok_4 := clientNodeMap[nodeId2]

			if !(ok_1 || ok_2) {
				log.Warning.Printf("Client ID: %s not present in the cluster\n", nodeId1)
				continue
			} else if !(ok_3 || ok_4) {
				log.Warning.Printf("Server ID: %s not present in the cluster\n", nodeId2)
				continue
			} else if ok_2 && ok_4 {
				log.Warning.Printf("Cannot breakConnection between %s and %s. Both are clients.\n", nodeId1, nodeId2)
				continue
			}

			var node1_break_conn_req cc.BreakConnectionRequest
			var node2_break_conn_req cc.BreakConnectionRequest
			var break_conn_reply cc.Nothing

			// between two servers
			if ok_1 && ok_3 {
				// info to send to server 1
				node1_break_conn_req.Id = nodeId2
				node1_break_conn_req.Address = "localhost:" + strconv.Itoa(serverNodeMap[nodeId2])

				// info to send to server 2
				node2_break_conn_req.Id = nodeId1
				node2_break_conn_req.Address = "localhost:" + strconv.Itoa(serverNodeMap[nodeId1])

				// make an RPC call to server 1 to send info about server 2
				node1_conn := getRPCConnection(node2_break_conn_req.Address)
				err1 := node1_conn.Call("ServerListener.BreakConnection",
						&node1_break_conn_req, &break_conn_reply)
				if err1 != nil {
					log.Warning.Println("ServerListener.BreakConnection RPC call failed.\n")
				}
				node1_conn.Close()

				// make an RPC call to server 2 to send info about server 1
				node2_conn := getRPCConnection(node1_break_conn_req.Address)
				err2 := node2_conn.Call("ServerListener.BreakConnection",
						&node2_break_conn_req, &break_conn_reply)
				if err2 != nil {
					log.Warning.Println("ServerListener.BreakConnection RPC call failed.\n")
				}
				node2_conn.Close()
			} else {
				// swap IDs so that nodeId1 is server and nodeId2 is client
				if ok_2 && ok_3 {
					nodeId1, nodeId2 = nodeId2, nodeId1
				}
				// client address
				clientAddress := "localhost:" + strconv.Itoa(clientNodeMap[nodeId2])

				// info to send to client
				node2_break_conn_req.Id = nodeId1
				node2_break_conn_req.Address = "localhost:" + strconv.Itoa(serverNodeMap[nodeId1])

				// make an RPC call to client to send info about server
				node2_conn := getRPCConnection(clientAddress)
				err := node2_conn.Call("ClientListener.BreakConnection",
						&node2_break_conn_req, &break_conn_reply)
				if err != nil {
					log.Warning.Println("ClientListener.BreakConnection RPC call failed.\n")
				}
				node2_conn.Close()

			}

		case "createConnection":
			log.Info.Println("Executing...", commandSplit)

			// get node IDs
			nodeId1 := commandSplit[1]
			nodeId2 := commandSplit[2]

			_, ok_1 := serverNodeMap[nodeId1]
			_, ok_2 := clientNodeMap[nodeId1]

			_, ok_3 := serverNodeMap[nodeId2]
			_, ok_4 := clientNodeMap[nodeId2]

			if !(ok_1 || ok_2) {
				log.Warning.Printf("Client ID: %s not present in the cluster\n", nodeId1)
				continue
			} else if !(ok_3 || ok_4) {
				log.Warning.Printf("Server ID: %s not present in the cluster\n", nodeId2)
				continue
			} else if ok_2 && ok_4 {
				log.Warning.Printf("Cannot breakConnection between %s and %s. Both are clients.\n", nodeId1, nodeId2)
				continue
			}

			var node1_create_conn_req cc.CreateConnectionRequest
			var node2_create_conn_req cc.CreateConnectionRequest
			var create_conn_reply cc.Nothing

			// between two servers
			if ok_1 && ok_3 {
				// info to send to server 1
				node1_create_conn_req.Id = nodeId2
				node1_create_conn_req.Address = "localhost:" + strconv.Itoa(serverNodeMap[nodeId2])

				// info to send to server 2
				node2_create_conn_req.Id = nodeId1
				node2_create_conn_req.Address = "localhost:" + strconv.Itoa(serverNodeMap[nodeId1])

				// make an RPC call to server 1 to send info about server 2
				node1_conn := getRPCConnection(node2_create_conn_req.Address)
				err1 := node1_conn.Call("ServerListener.CreateConnection",
						&node1_create_conn_req, &create_conn_reply)
				if err1 != nil {
					log.Warning.Println("ServerListener.CreateConnection RPC call failed.\n")
				}
				node1_conn.Close()

				// make an RPC call to server 2 to send info about server 1
				node2_conn := getRPCConnection(node1_create_conn_req.Address)
				err2 := node2_conn.Call("ServerListener.CreateConnection",
						&node2_create_conn_req, &create_conn_reply)
				if err2 != nil {
					log.Warning.Println("ServerListener.CreateConnection RPC call failed.\n")
				}
				node2_conn.Close()
			} else {
				// swap IDs so that nodeId1 is server and nodeId2 is client
				if ok_2 && ok_3 {
					nodeId1, nodeId2 = nodeId2, nodeId1
				}
				// client address
				clientAddress := "localhost:" + strconv.Itoa(clientNodeMap[nodeId2])

				// info to send to client
				node2_create_conn_req.Id = nodeId1
				node2_create_conn_req.Address = "localhost:" + strconv.Itoa(serverNodeMap[nodeId1])

				// make an RPC call to client to send info about server
				node2_conn := getRPCConnection(clientAddress)
				err := node2_conn.Call("ClientListener.CreateConnection",
						&node2_create_conn_req, &create_conn_reply)
				if err != nil {
					log.Warning.Println("ClientListener.CreateConnection RPC call failed.\n")
				}
				node2_conn.Close()

			}

		case "stabilize":
			log.Info.Println("TODO ", commandSplit)
		case "printStore":
			log.Info.Println("TODO ", commandSplit)

		case "put":
			log.Info.Println("Executing...", commandSplit)

			// get client ID
			client_id := commandSplit[1]
			// check if client present
			if _, ok := clientNodeMap[client_id]; ok == false {
				log.Warning.Printf("Client [ID: %s] is not present in the cluster.\n", client_id)
				continue
			}
			clientPort := clientNodeMap[client_id]
			rpc_client := getRPCConnection("localhost:" + strconv.Itoa(clientPort))
			if rpc_client != nil {
				var put_kv_client_req cc.PutKVClientRequest
				var put_kv_client_reply cc.Nothing

				put_kv_client_req.Key = commandSplit[2]
				put_kv_client_req.Value = commandSplit[3]

				err := rpc_client.Call("ClientListener.PutKVClient",
					&put_kv_client_req, &put_kv_client_reply)

				if err != nil {
					log.Warning.Printf("PutKeyValueClient request to client [ID: %s] failed [err: %s].", client_id, err)
				}
				rpc_client.Close()
			} else {
				log.Warning.Println("getRPCConnection returned a nil value.")
			}

		case "get":
			log.Info.Println("Executing...", commandSplit)

			// get client ID
			client_id := commandSplit[1]
			// check if client present
			if _, ok := clientNodeMap[client_id]; ok == false {
				log.Warning.Printf("Client [ID: %s] is not present in the cluster.\n", client_id)
				continue
			}
			clientPort := clientNodeMap[client_id]
			rpc_client := getRPCConnection("localhost:" + strconv.Itoa(clientPort))
			if rpc_client != nil {
				var get_kv_client_req cc.GetKVClientRequest
				var get_kv_client_reply cc.GetKVClientReply

				get_kv_client_req.Key = commandSplit[2]

				err := rpc_client.Call("ClientListener.GetKVClient",
					&get_kv_client_req, &get_kv_client_reply)

				if err != nil {
					log.Warning.Printf("PutKeyValueClient request to client [ID: %s] failed [err: %s].", client_id, err)
				}
				rpc_client.Close()
				log.Info.Printf("Key: %s, Value: %s, Version: %f.\n",
					get_kv_client_reply.Key,
					get_kv_client_reply.Value,
					get_kv_client_reply.Version,
				)
			} else {
				log.Warning.Println("getRPCConnection returned a nil value.")
			}

		case "printMemberList":
			log.Info.Println("Executing...", commandSplit)
			// get server ID
			server_id := commandSplit[1]
			// check if server present
			if _, ok := serverNodeMap[server_id]; ok == false {
				log.Warning.Printf("Server [ID: %s] is not present in the cluster.\n", server_id)
				continue
			}
			serverPort := serverNodeMap[server_id]
			rpc_client := getRPCConnection("localhost:" + strconv.Itoa(serverPort))
			if rpc_client != nil {
				var print_member_list_req cc.Nothing
				var print_member_list_reply cc.Nothing

				err := rpc_client.Call("ServerListener.PrintMembershipList",
					&print_member_list_req, &print_member_list_reply)

				if err != nil {
					log.Warning.Printf("PrintMembershipList request to server [ID: %s] failed [err: %s].", server_id, err)
				}
				rpc_client.Close()
			} else {
				log.Warning.Println("getRPCConnection returned a nil value.")
			}

		default:
			log.Warning.Println("Command", commandSplit, "not recognized")
			break
		}
	}
	if err := scanner.Err(); err != nil {
		log.Panic.Panicln("Standard input error:", err)
	}
}
