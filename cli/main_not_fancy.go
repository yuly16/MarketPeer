package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/blockchain/account"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/udp"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

//type nodeConfig struct {
//	Addr       string
//	Balance    int
//	Storage    map[string]interface{}
//	PrivateKey string
//}
//
//type config struct {
//	Nodes []nodeConfig
//}
//
//// based on init accounts info, generate the genesis block
//func generateGenesisBlock(kvFactory storage.KVFactory, accounts ...*account.Account) *block.Block {
//	bb := block.NewBlockBuilder(kvFactory).
//		SetParentHash(block.DUMMY_PARENT_HASH).
//		SetNonce(0).
//		SetNumber(0).
//		SetDifficulty(2).
//		SetBeneficiary(*account.NewAddress([8]byte{}))
//	for _, acc := range accounts {
//		bb.SetAddrState(acc.GetAddr(), acc.GetState())
//	}
//	b := bb.Build()
//	return b
//}
//
//func initBlockchain(c *config) *block.Block {
//
//	accounts := make([]*account.Account, 0)
//	for _, node := range c.Nodes {
//		pri, err := crypto.HexToECDSA(node.PrivateKey)
//		if err != nil {
//			panic(err)
//		}
//		pub := &pri.PublicKey
//		acb := account.NewAccountBuilder(crypto.FromECDSAPub(pub), storage.CreateSimpleKV).
//			WithBalance(uint(node.Balance))
//		for k, v := range node.Storage {
//			acb.WithKV(k, v)
//		}
//		ac := acb.Build()
//		accounts = append(accounts, ac)
//
//	}
//	return generateGenesisBlock(storage.CreateSimpleKV, accounts...)
//}
//
//func nodeConf(c *config, addr string) *nodeConfig {
//	for _, node := range c.Nodes {
//		if node.Addr == addr {
//			return &node
//		}
//	}
//	return nil
//}
//
//func initPeers(c *config, me string) []string {
//	ret := make([]string, 0)
//	for _, nodeC := range c.Nodes {
//		if me == nodeC.Addr {
//			continue
//		}
//		ret = append(ret, nodeC.Addr)
//	}
//	return ret
//}

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

	// read from command
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("> ")
		paramsS, err := reader.ReadString('\n')
		paramsS = strings.TrimRight(paramsS, "\n")
		if err != nil {
			fmt.Println(err)
		}
		params := strings.Split(paramsS, " ")
		action := params[0]
		if action == "AddPeer" {
			addr := params[1]
			fmt.Println(addr)
			clientNode.AddPeers(addr)
		} else if action == "Transfer" {
			value, _ := strconv.Atoi(params[1])
			to := params[2]
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
		} else if action == "UploadProduct" {
			product := params[1]
			amount, _ := strconv.Atoi(params[2])
			if err := clientNode.BlockChainFullNode.UploadProduct(product, amount); err != nil {
				fmt.Println("upload product fail: ", err)
			} else {
				fmt.Println("upload product success, new account state:")
				fmt.Println(clientNode.BlockChainFullNode.ShowAccount().String())
			}
		} else if action == "ShowAccount" {
			info := clientNode.BlockChainFullNode.ShowAccount()
			fmt.Println("account info: " + info.String())
		} else if action == "ShowChain" {
			fmt.Println(clientNode.BlockChainFullNode.GetChain().String())
		} else {
			fmt.Println("unknown action: ", action)
		}
	}

	//clientNode.AddPeers("127.0.0.1:0")
	//time.Sleep(time.Second * 5)
	//
	//time.Sleep(time.Second * 1000)
	clientNode.Stop()
}
