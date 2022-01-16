package main

import (
	"bufio"
	"fmt"
	"go.dedis.ch/cs438/client/client"
	"os"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	// init a new Client Node
	cli := client.NewClient(nil, nil, "")
	for {
		paramsS, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}
		params := strings.Split(paramsS, ",")
		action := params[0]
		if action == "AddPeer" {
			addr := params[1]
			cli.AddPeers(addr)
		}
		//if action == "TransferEpfer" {
		//	to := params[1]
		//	amount := params[2]
		//}
		//else if action == "CreateContract" {
		//	codeFile := params[1]
		//} else if action == "ActivateContract" {
		//
		//}
	}
}
