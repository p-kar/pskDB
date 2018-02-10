package main

import (
	"./logger"
	"bufio"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	log := logger.NewLogger("[  MASTER  ] ", os.Stdout, os.Stdout,
		os.Stdout, os.Stderr, os.Stderr)
	// map from node(server) id to its port number
	serverNodeMap := make(map[string]int)
	// map from node(client) id to its port number
	clientNodeMap := make(map[string]int)

	// Starting value of the port numbers used by servers
	serverNextPort := 9000
	// map from node(client) id to its port number
	// clientNodeMap := make(map[string]int)
	// Starting value of the port numbers used by clients
	// clientNextPort := 10000

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
				client := getRPCConnection("localhost:" + strconv.Itoa(serverNodeMap[nodeId]))
				if client != nil {
					client.Call("ServerListener.Ping", &req, &reply)
					if reply == req {
						log.Info.Println("Started server with process id", joinCmd.Process.Pid)
						break
					}
				}
				if calls > 100 {
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
			}

		case "killServer":
			log.Info.Println("Executing...", commandSplit)
			// get Node ID
			nodeId := commandSplit[1]
			// get server port number for RPC call
			if _, ok := serverNodeMap[nodeId]; ok == false {
				log.Warning.Printf("Server ID: %s is not present in the cluster\n", nodeId)
				continue
			}
			serverPort := serverNodeMap[nodeId]
			client := getRPCConnection("localhost:" + strconv.Itoa(serverPort))
			if client != nil {
				req := true
				var reply bool
				client.Call("ServerListener.KillServer", &req, &reply)
				// log.Info.Println("killServer finished")
				delete(serverNodeMap, nodeId)
				client.Close()
			} else {
				log.Warning.Println("getRPCConnection returned a null value")
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
			args := []string{}
			args = append(args, clientNodeId, strconv.Itoa(serverNextPort), strconv.Itoa(serverNodeMap[serverNodeId]))
			clientNodeMap[clientNodeId] = serverNextPort
			serverNextPort++

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
			//client := getRPCConnection("localhost:" + strconv.Itoa(clientNodeMap[clientNodeId]))
			for {
				calls++
				client := getRPCConnection("localhost:" + strconv.Itoa(clientNodeMap[clientNodeId]))
				if client != nil {
					client.Call("ClientListener.Ping", &req, &reply)
					if reply == req {
						log.Info.Println("Started client with process id", joinCmd.Process.Pid)
						break
					}
				}
				if calls > 1000 {
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
			}

		case "breakConnection":
			log.Info.Println("Executing...", commandSplit)

			// get server and client IDs
			clientNodeId := commandSplit[1]
			serverNodeId := commandSplit[2]

			var firstIsClient bool
			var secondIsClient bool

			_, ok_1 := serverNodeMap[clientNodeId]
			_, ok_2 := clientNodeMap[clientNodeId]

			_, ok_3 := serverNodeMap[serverNodeId]
			_, ok_4 := clientNodeMap[serverNodeId]

			if !(ok_1 || ok_2) {
				log.Warning.Printf("Client ID: %s not present in the cluster\n", clientNodeId)
				continue
			} else if !(ok_3 || ok_4) {
				log.Warning.Printf("Server ID: %s not present in the cluster\n", serverNodeId)
				continue
			}
			// } else if _, ok := serverNodeMap[clientNodeId]; ok == false {
			//  log.Warning.Printf("Server ID: %s not present in the cluster\n", clientNodeId)
			//  continue
			// }

			var serverBlackListInfo BlackListInfo
			var scBlackListInfo BlackListInfo

			// fill second node info
			serverBlackListInfo.Id = serverNodeId
			if _, ok := serverNodeMap[serverNodeId]; ok == true {
				secondIsClient = false
				serverBlackListInfo.Port_num = strconv.Itoa(serverNodeMap[serverNodeId])
			} else {
				secondIsClient = true
				serverBlackListInfo.Port_num = strconv.Itoa(clientNodeMap[serverNodeId])
			}
			serverBlackListInfo.IP_address = "localhost"

			// fill first node info

			scBlackListInfo.Id = clientNodeId
			if _, ok := serverNodeMap[clientNodeId]; ok == true {
				firstIsClient = false
				scBlackListInfo.Port_num = strconv.Itoa(serverNodeMap[clientNodeId])
			} else {
				firstIsClient = true
				scBlackListInfo.Port_num = strconv.Itoa(clientNodeMap[clientNodeId])
			}
			scBlackListInfo.IP_address = "localhost"

			server := getRPCConnection(serverBlackListInfo.IP_address + ":" + serverBlackListInfo.Port_num)
			var reply_server Nothing
			if server != nil {
				if secondIsClient == false {
					err := server.Call("ServerListener.BreakConnection", &scBlackListInfo, &reply_server)
					if err != nil {
						log.Warning.Printf("RPC call to server at port number: %s failed.\n", serverBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent BreakConnection to server [ID:%s, Port_num:%s].\n", serverBlackListInfo.Id, serverBlackListInfo.Port_num)
				} else {
					err := server.Call("ClientListener.BreakConnection", &scBlackListInfo, &reply_server)
					if err != nil {
						log.Warning.Printf("RPC call to client at port number: %s failed.\n", serverBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent BreakConnection to client [ID:%s, Port_num:%s].\n", serverBlackListInfo.Id, serverBlackListInfo.Port_num)
				}
			}
			server.Close()

			sc := getRPCConnection(scBlackListInfo.IP_address + ":" + scBlackListInfo.Port_num)
			var reply_sc Nothing
			if sc != nil {
				if firstIsClient == false {
					err := sc.Call("ServerListener.BreakConnection", &serverBlackListInfo, &reply_sc)
					if err != nil {
						log.Warning.Printf("RPC call to server at port number: %s failed.\n", scBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent BreakConnection to server [ID:%s, Port_num:%s].\n", scBlackListInfo.Id, scBlackListInfo.Port_num)
				} else {
					err := sc.Call("ClientListener.BreakConnection", &serverBlackListInfo, &reply_sc)
					if err != nil {
						log.Warning.Printf("RPC call to server at port number: %s failed.\n", scBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent BreakConnection to client [ID:%s, Port_num:%s].\n", scBlackListInfo.Id, scBlackListInfo.Port_num)
				}

			}
			sc.Close()
		case "createConnection":
			log.Info.Println("Executing...", commandSplit)

			// get server and client IDs
			clientNodeId := commandSplit[1]
			serverNodeId := commandSplit[2]

			var firstIsClient bool
			var secondIsClient bool

			_, ok_1 := serverNodeMap[clientNodeId]
			_, ok_2 := clientNodeMap[clientNodeId]

			_, ok_3 := serverNodeMap[serverNodeId]
			_, ok_4 := clientNodeMap[serverNodeId]

			if !(ok_1 || ok_2) {
				log.Warning.Printf("Client ID: %s not present in the cluster\n", clientNodeId)
				continue
			} else if !(ok_3 || ok_4) {
				log.Warning.Printf("Server ID: %s not present in the cluster\n", serverNodeId)
				continue
			}
			// } else if _, ok := serverNodeMap[clientNodeId]; ok == false {
			//  log.Warning.Printf("Server ID: %s not present in the cluster\n", clientNodeId)
			//  continue
			// }

			var serverBlackListInfo BlackListInfo
			var scBlackListInfo BlackListInfo

			// fill second node info
			serverBlackListInfo.Id = serverNodeId
			if _, ok := serverNodeMap[serverNodeId]; ok == true {
				secondIsClient = false
				serverBlackListInfo.Port_num = strconv.Itoa(serverNodeMap[serverNodeId])
			} else {
				secondIsClient = true
				serverBlackListInfo.Port_num = strconv.Itoa(clientNodeMap[serverNodeId])
			}
			serverBlackListInfo.IP_address = "localhost"

			// fill first node info

			scBlackListInfo.Id = clientNodeId
			if _, ok := serverNodeMap[clientNodeId]; ok == true {
				firstIsClient = false
				scBlackListInfo.Port_num = strconv.Itoa(serverNodeMap[clientNodeId])
			} else {
				firstIsClient = true
				scBlackListInfo.Port_num = strconv.Itoa(clientNodeMap[clientNodeId])
			}
			scBlackListInfo.IP_address = "localhost"

			server := getRPCConnection(serverBlackListInfo.IP_address + ":" + serverBlackListInfo.Port_num)
			var reply_server Nothing
			if server != nil {
				if secondIsClient == false {
					err := server.Call("ServerListener.CreateConnection", &scBlackListInfo, &reply_server)
					if err != nil {
						log.Warning.Printf("RPC call to server at port number: %s failed.\n", serverBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent CreateConnection to server [ID:%s, Port_num:%s].\n", serverBlackListInfo.Id, serverBlackListInfo.Port_num)
				} else {
					err := server.Call("ClientListener.CreateConnection", &scBlackListInfo, &reply_server)
					if err != nil {
						log.Warning.Printf("RPC call to client at port number: %s failed.\n", serverBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent CreateConnection to client [ID:%s, Port_num:%s].\n", serverBlackListInfo.Id, serverBlackListInfo.Port_num)
				}
			}
			server.Close()

			sc := getRPCConnection(scBlackListInfo.IP_address + ":" + scBlackListInfo.Port_num)
			var reply_sc Nothing
			if sc != nil {
				if firstIsClient == false {
					err := sc.Call("ServerListener.CreateConnection", &serverBlackListInfo, &reply_sc)
					if err != nil {
						log.Warning.Printf("RPC call to server at port number: %s failed.\n", scBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent CreateConnection to server [ID:%s, Port_num:%s].\n", scBlackListInfo.Id, scBlackListInfo.Port_num)
				} else {
					err := sc.Call("ClientListener.CreateConnection", &serverBlackListInfo, &reply_sc)
					if err != nil {
						log.Warning.Printf("RPC call to server at port number: %s failed.\n", scBlackListInfo.Port_num)
						continue
					}
					log.Info.Printf("Sent CreateConnection to client [ID:%s, Port_num:%s].\n", scBlackListInfo.Id, scBlackListInfo.Port_num)
				}

			}
			sc.Close()
		case "stabilize":
			log.Info.Println("TODO ", commandSplit)
		case "printStore":
			log.Info.Println("TODO ", commandSplit)
		case "put":
			log.Info.Println("TODO ", commandSplit)
		case "get":
			log.Info.Println("TODO ", commandSplit)
		default:
			log.Warning.Println("Command", commandSplit, "not recognized")
			break
		}
	}
	if err := scanner.Err(); err != nil {
		log.Panic.Panicln("Standard input error:", err)
	}
}
