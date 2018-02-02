package main

import (
    "fmt"
    "bufio"
    "os"
    "strings"
    "os/exec"
    "log"
    "strconv"
)

func main() {
    scanner := bufio.NewScanner(os.Stdin)

    nodeMap := make(map[string]int) // map from node(server or client) id to its port number
                                    // nodeId is a string
    nextPort := 9000   // Starting value of the port numbers used by nodes

    for scanner.Scan(){
        // read and process the command from stdin
        command := scanner.Text()
        commandSplit := strings.Split(command, " ")

        switch commandSplit[0] {
            case "joinServer":
                log.Println("Executing...", commandSplit)
                
                // Create startup info i.e., arguments
                // example arguments ["1" "9001" "0" "9004" "2" "9003"]
                nodeId := commandSplit[1]
                args := []string{}
                args = append(args, nodeId, strconv.Itoa(nextPort))
                for id, port := range nodeMap {
                    args = append(args, id, strconv.Itoa(port))
                }
                nodeMap[nodeId] = nextPort
                nextPort++

                // spawn the server
                joinCmd := exec.Command("./serverNode", args...)
                joinCmd.Stdout = os.Stdout
                joinCmd.Stderr = os.Stderr
                err := joinCmd.Start()
                if err != nil {
                    fmt.Println(err)
                }
                log.Println("Started server with process id", joinCmd.Process.Pid)
                go joinCmd.Wait()
            case "killServer":
                fmt.Println("TODO ", commandSplit)
            case "joinClient":
                fmt.Println("TODO ", commandSplit)
            case "breakConnection":
                fmt.Println("TODO ", commandSplit)
            case "createConnection":
                fmt.Println("TODO ", commandSplit)
            case "stabilize":
                fmt.Println("TODO ", commandSplit)
            case "printStore":
                fmt.Println("TODO ", commandSplit)
            case "put":
                fmt.Println("TODO ", commandSplit)
            case "get":
                fmt.Println("TODO ", commandSplit)
        }
    }
    if err := scanner.Err(); err != nil {
        fmt.Fprintln(os.Stderr, "reading standard input:", err)
    }
}