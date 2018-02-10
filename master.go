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
			log.Info.Println("Started server with process id", joinCmd.Process.Pid)
			go joinCmd.Wait()
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
			log.Info.Println("TODO ", commandSplit)
		case "breakConnection":
			log.Info.Println("TODO ", commandSplit)
		case "createConnection":
			log.Info.Println("TODO ", commandSplit)
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
