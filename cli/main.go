package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/blockchain/account"
	"go.dedis.ch/cs438/blockchain/block"
	"go.dedis.ch/cs438/blockchain/storage"
	"go.dedis.ch/cs438/client/client"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/udp"

	// "os"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
)

type nodeConfig struct {
	Addr       string
	Balance    int
	Storage    map[string]interface{}
	PrivateKey string
}

type config struct {
	Nodes []nodeConfig
}

// based on init accounts info, generate the genesis block
func generateGenesisBlock(kvFactory storage.KVFactory, accounts ...*account.Account) *block.Block {
	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash(block.DUMMY_PARENT_HASH).
		SetNonce(0).
		SetNumber(0).
		SetDifficulty(2).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	for _, acc := range accounts {
		bb.SetAddrState(acc.GetAddr(), acc.GetState())
	}
	b := bb.Build()
	return b
}

func initBlockchain(c *config) *block.Block {

	accounts := make([]*account.Account, 0)
	for _, node := range c.Nodes {
		pri, err := crypto.HexToECDSA(node.PrivateKey)
		if err != nil {
			panic(err)
		}
		pub := &pri.PublicKey
		acb := account.NewAccountBuilder(crypto.FromECDSAPub(pub), storage.CreateSimpleKV).
			WithBalance(uint(node.Balance))
		for k, v := range node.Storage {
			acb.WithKV(k, v)
		}
		ac := acb.Build()
		accounts = append(accounts, ac)

	}
	return generateGenesisBlock(storage.CreateSimpleKV, accounts...)
}

func nodeConf(c *config, addr string) *nodeConfig {
	for _, node := range c.Nodes {
		if node.Addr == addr {
			return &node
		}
	}
	return nil
}

func initPeers(c *config, me string) []string {
	ret := make([]string, 0)
	for _, nodeC := range c.Nodes {
		if me == nodeC.Addr {
			continue
		}
		ret = append(ret, nodeC.Addr)
	}
	return ret
}

type nodeConfig struct {
	Addr       string
	Balance    int
	Storage    map[string]interface{}
	PrivateKey string
}

type config struct {
	Nodes []nodeConfig
}

// based on init accounts info, generate the genesis block
func generateGenesisBlock(kvFactory storage.KVFactory, accounts ...*account.Account) *block.Block {
	bb := block.NewBlockBuilder(kvFactory).
		SetParentHash(block.DUMMY_PARENT_HASH).
		SetNonce(0).
		SetNumber(0).
		SetDifficulty(2).
		SetBeneficiary(*account.NewAddress([8]byte{}))
	for _, acc := range accounts {
		bb.SetAddrState(acc.GetAddr(), acc.GetState())
	}
	b := bb.Build()
	return b
}

func initBlockchain(c *config) *block.Block {

	accounts := make([]*account.Account, 0)
	for _, node := range c.Nodes {
		pri, err := crypto.HexToECDSA(node.PrivateKey)
		if err != nil {
			panic(err)
		}
		pub := &pri.PublicKey
		acb := account.NewAccountBuilder(crypto.FromECDSAPub(pub), storage.CreateSimpleKV).
			WithBalance(uint(node.Balance))
		for k, v := range node.Storage {
			acb.WithKV(k, v)
		}
		ac := acb.Build()
		accounts = append(accounts, ac)

	}
	return generateGenesisBlock(storage.CreateSimpleKV, accounts...)
}

func nodeConf(c *config, addr string) *nodeConfig {
	for _, node := range c.Nodes {
		if node.Addr == addr {
			return &node
		}
	}
	return nil
}

func initPeers(c *config, me string) []string {
	ret := make([]string, 0)
	for _, nodeC := range c.Nodes {
		if me == nodeC.Addr {
			continue
		}
		ret = append(ret, nodeC.Addr)
	}
	return ret
}

func main() {
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	c := &config{}
	if err := json.Unmarshal(data, c); err != nil {
		panic(err)
	}
	genesis := initBlockchain(c)
	argsWithoutProg := os.Args[1:]
	addr := argsWithoutProg[0]
	nodeC := nodeConf(c, addr)
	if nodeC == nil {
		fmt.Printf("addr=%s not in config.json, exit\n", addr)
	}
	// initialize
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	c := &config{}
	if err := json.Unmarshal(data, c); err != nil {
		panic(err)
	}
	genesis := initBlockchain(c)
	argsWithoutProg := os.Args[1:]
	addr := argsWithoutProg[0]
	nodeC := nodeConf(c, addr)
	if nodeC == nil {
		fmt.Printf("addr=%s not in config.json, exit\n", addr)
	}
	// initialize
	transp := udp.NewUDP()
	bitNum := 12
	socket, _ := transp.CreateSocket(addr)
	privateKey, _ := crypto.HexToECDSA(nodeC.PrivateKey)
	clientNode := *z.NewClient(nil,
		z.WithSocket(socket),
		z.WithMessageRegistry(standard.NewRegistry()),
		z.WithPrivateKey(privateKey),
		z.WithGenesisBlock(genesis),

		z.WithHeartbeat(time.Millisecond*500),
		z.WithAntiEntropy(time.Millisecond*100),
		z.WithChordBits(uint(bitNum)),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*250))
	neis := initPeers(c, addr)
	for _, nei := range neis {
		clientNode.AddPeers(nei)
	}
	clientNode.Start()
	fmt.Printf("network addr=%s, account addr=%s\n", clientNode.BlockChainFullNode.GetAddr(),
		clientNode.BlockChainFullNode.GetAccountAddr().String())
	fmt.Printf("peers=%v\n", neis)
	fmt.Println("initial block: ")
	fmt.Println(clientNode.BlockChainFullNode.GetChain().String())
	time.Sleep(time.Second * 2)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Type Init to init chord, type Join to join chord:")
		params, _ := reader.ReadString('\n')
		params = strings.TrimRight(params, "\n")
		if params == "Init" {
			if c.Nodes[0].Addr == addr {
				clientNode.ChordNode.Init(c.Nodes[1].Addr)
				break
			} else if c.Nodes[1].Addr == addr {
				clientNode.ChordNode.Init(c.Nodes[0].Addr)
				break
			} else {
				fmt.Println("This node should input Join.")
			}
		} else if params == "Join" {
			if c.Nodes[0].Addr == addr {
				fmt.Println("This node should input Init.")
			} else if c.Nodes[1].Addr == addr {
				fmt.Println("This node should input Init.")
			} else {
				err := clientNode.ChordNode.Join(c.Nodes[0].Addr)
				if err != nil {
					return
				}
				break
			}
		} else {
			fmt.Println("wrong format. ")
		}
	}

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
	transfer_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}
	showaccount_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}
	showchain_validate := func(input string) error {
		fmt.Println("Here is your input command", 1+1)
		return nil
	}
	// Front-end CLI UI
	for {
		cmd_prompt := promptui.Select{
			Label: "Select your command:",
			Items: []string{"View Products", "Input Product Information", "Transfer", "ShowAccount", "ShowChain", "Exit"},
		}

		is_exit := false
		_, cmd, err := cmd_prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		addpeer_prompt := promptui.Prompt{
			Label:    "[Add peer] input a valid IP address: ",
			Validate: addpeer_validate,
		}
		initchord_prompt := promptui.Prompt{
			Label:    "[Init Chord] input a valid IP address: ",
			Validate: initchord_validate,
		}
		joinchord_prompt := promptui.Prompt{
			Label:    "[Join Chord] input a valid IP address: ",
			Validate: joinchord_validate,
		}
		inputproduct_prompt := promptui.Prompt{
			Label:    "[Input Product] input product information: ",
			Validate: inputproduct_validate,
		}
		viewproduct_prompt := promptui.Prompt{
			Label:    "[View Product] input product name: ",
			Validate: viewproduct_validate,
		}
		transfer_prompt := promptui.Prompt{
			Label:    "[Transfer] input transfer (format: Value To): ",
			Validate: transfer_validate,
		}
		showaccount_prompt := promptui.Prompt{
			Label:    "[Show Account]",
			Validate: showaccount_validate,
		}
		showchain_prompt := promptui.Prompt{
			Label:    "[Show Chain]",
			Validate: showchain_validate,
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
						Name:  name,
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
		case "Transfer":
			info, _ := transfer_prompt.Run()
			params := strings.Split(info, " ")
			value, _ := strconv.Atoi(params[0])
			to := params[1]
			fmt.Println(value, to)
			addrBytesSlice, _ := hex.DecodeString(to)
			addrBytes := [8]byte{}
			copy(addrBytes[:], addrBytesSlice)
			if err := clientNode.BlockChainFullNode.TransferEpfer(account.Address{addrBytes, to}, value); err != nil {
				fmt.Println("transfer fail: ", err)
			} else {
				fmt.Println("transfer success, new account state:")
				fmt.Println(clientNode.BlockChainFullNode.ShowAccount().String())
			}
		case "ShowAccount":
			_, _ = showaccount_prompt.Run()
			info := clientNode.BlockChainFullNode.ShowAccount()
			fmt.Println("account info: " + info.String())
		case "ShowChain":
			_, _ = showchain_prompt.Run()
			fmt.Println(clientNode.BlockChainFullNode.GetChain().String())
			// execution
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
