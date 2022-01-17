package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/client/client"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/udp"
    "io/ioutil"
	"strings"
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
	// Claim validation function for each command
	initchord_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}
	// Claim validation function for each command
	joinchord_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}
	inputproduct_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}
	viewproduct_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}
	// format: (contract name, acceptor address)
	propose_contract_validate := func(input string) error {
		params := strings.Split(input, " ")
		if len(params) != 2 {
			return fmt.Errorf("propose contract parameters not met")
		}
		return nil
	}

	// Front-end CLI UI
	for {
		cmd_prompt := promptui.Select{
			Label: "Select your command:",
			Items: []string{"Add Peer", "Init Chord", "Join Chord", "View Products", "Input Product Information", "Propose Smart Contract", "Exit"},
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
		initchord_prompt := promptui.Prompt{
			Label:	"[Init Chord] input a valid IP address: ",
			Validate: initchord_validate,
		}
		joinchord_prompt := promptui.Prompt{
			Label:	"[Join Chord] input a valid IP address: ",
			Validate: joinchord_validate,
		}
		inputproduct_prompt := promptui.Prompt{
			Label:	"[Input Product] input product information: \n" +
				"format: Name Owner \n" +
				"example: orange 127.0.0.1",
			Validate: inputproduct_validate,
		}
		viewproduct_prompt := promptui.Prompt{
			Label:	"[View Product] input product name: ",
			Validate: viewproduct_validate,
		}
		propose_contract_prompt := promptui.Prompt{
			Label: "[Propose contract] input (contract name, acceptor address): ",
			Validate: propose_contract_validate,
		}

		switch cmd {
		case "Add Peer":
			address, err := addpeer_prompt.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			} else {
				clientNode.AddPeers(address)
				fmt.Println("Add a neighbour successful. ")
			}
		case "Init Chord":
			address, err := initchord_prompt.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			} else {
				clientNode.ChordNode.Init(address)
				fmt.Println("Init chord successful. ")
			}
		case "Join Chord":
			address, err := joinchord_prompt.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			} else {
				clientNode.ChordNode.Join(address)
				fmt.Println("Join chord successful. ")
			}
		case "Input Product Information":
			info, err := inputproduct_prompt.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			} else {
				infos := strings.Split(info, " ")
				if len(infos) != 2 {
					fmt.Println("Wrong format.")
				} else {
					name := infos[0]
					address := infos[1]
					product := client.Product{
						Name: name,
						Owner: address,
					}
					err := clientNode.StoreProductString(name, product)
					if err != nil {
						fmt.Println("store product error")
						fmt.Println(err)
						return
					}
				}
			}
		case "View Products":
			info, err := viewproduct_prompt.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return
			} else {
				product, exist := clientNode.ReadProductString(info)
				if exist {
					fmt.Printf("ProductInfo: name: %s, address: %s\n", product.Name, product.Owner)
				} else {
					fmt.Println("the product doesn't exist in chord. ")
				}
			}
		case "Propose Smart Contract":
			result, err := propose_contract_prompt.Run()
			if err != nil {
				fmt.Println("Invalid input form: %v", err)
				break
			}
			
			// call by cli params
			params := strings.Split(result, " ")
			contract_name := params[0]
			contract_filefolder_path := "../contract_files/"
			contract_file_path := contract_filefolder_path + contract_name
			acceptor_address := params[1]

			contract_file_content, err := ioutil.ReadFile(contract_file_path)
			if err != nil {
				fmt.Println("Invalid file path: %w", err)
				break
			}

			contract_addr, err := clientNode.CreateContract(contract_name, string(contract_file_content), acceptor_address)
			if err != nil {
				fmt.Println("Fail to create contract: %v", err)
				break
			}
			fmt.Println("Contract created at account: ", contract_addr)

		case "Exit":
			clientNode.Stop()
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