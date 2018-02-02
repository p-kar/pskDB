package main

import "os"
import "fmt"
import "log"

type NodeInfo struct {
	Id string
	Port string
}

type Node struct {
	MyInfo NodeInfo
}

func (n Node) String() string {
	return fmt.Sprintf("Id : %v, Port : %v", n.MyInfo.Id, n.MyInfo.Port)
}

func main() {
	fmt.Println(os.Args)
	n := Node{
			NodeInfo{os.Args[1], os.Args[2]},
		 }
	log.Println(n)
    for {
    }
}