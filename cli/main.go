package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/udp"
	// "os"
	// "strings"
	"time"

	"github.com/manifoldco/promptui"
)

func main() {
	// initialize
	transp := udp.NewUDP()
	bitNum := 12
	socket, _ := transp.CreateSocket("127.0.0.1:0")
	privateKey, _ := crypto.GenerateKey()
	clientNode := *z.NewClient(nil,
		z.WithSocket(socket),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey),
		z.WithHeartbeat(time.Millisecond*500),
		z.WithAntiEntropy(time.Millisecond*100),
		z.WithChordBits(uint(bitNum)),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*250))
	clientNode.Start()
	time.Sleep(time.Second * 2)

	// read from command
	// reader := bufio.NewReader(os.Stdin)
	// for {
	// 	fmt.Println("Waiting for input...")
	// 	paramsS, err := reader.ReadString('\n')
	// 	paramsS = strings.TrimRight(paramsS, "\n")
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	params := strings.Split(paramsS, " ")
	// 	action := params[0]
	// 	if action == "AddPeer" {
	// 		addr := params[1]
	// 		fmt.Println(addr)
	// 		clientNode.AddPeers(addr)
	// 	}
	// }
	
	// Claim validation function for each command
	addpeer_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}

	// Front-end CLI UI
	for {
		cmd_prompt := promptui.Select{
			Label: "Select your command:",
			Items: []string{"Add Peer", "View Products", "Input Product Information", "Exit"},
		}

		is_exit := false
		_, cmd, err := cmd_prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		addpeer_prompt := promptui.Prompt{
			Label:	"[Add peer] input a valid IP address: ",
			Validate: addpeer_validate,
		}

		switch cmd {
		case "Add Peer":
			_, err := addpeer_prompt.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			}
			// execution
		case "Exit":
			is_exit = true
		}

		if is_exit {
			break
		}
	}

	//clientNode.AddPeers("127.0.0.1:0")
	//time.Sleep(time.Second * 5)
	//
	//time.Sleep(time.Second * 1000)
	clientNode.Stop()
}
