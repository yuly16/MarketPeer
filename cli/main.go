package main

import (
	"bufio"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"go.dedis.ch/cs438/client/client"
	z "go.dedis.ch/cs438/internal/testing"
	"go.dedis.ch/cs438/registry/standard"
	"go.dedis.ch/cs438/transport/udp"
	"os"
	"strings"
	"time"
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
		z.WithAntiEntropy(time.Second*1),
		z.WithChordBits(uint(bitNum)),
		z.WithStabilizeInterval(time.Millisecond*500),
		z.WithFixFingersInterval(time.Millisecond*250))
	clientNode.Start()
	time.Sleep(time.Second * 2)

	// read from command
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Waiting for input...")
		paramsS, err := reader.ReadString('\n')
		paramsS = strings.TrimRight(paramsS, "\n")
		if err != nil {
			fmt.Println(err)
		}
		params := strings.Split(paramsS, " ")
		action := params[0]
		if action == "addpeer" {
			addr := params[1]
			fmt.Println(addr)
			clientNode.AddPeers(addr)
			fmt.Println("Add a neighbour successful. ")
		} else if action == "initchordsystem" {
			addr := params[1]
			clientNode.ChordNode.Init(addr)
			fmt.Println("Init chord successful. ")
		} else if action == "addchordsystem" {
			addr := params[1]
			err := clientNode.ChordNode.Join(addr)
			if err != nil {
				fmt.Println("add chord system error")
				fmt.Println(err)
				return
			}
			fmt.Println("Add a chord system successful. ")
		} else if action == "storeproduct" {
			productName := params[1]
			ownerAddress := params[2]
			product := client.Product{Name: productName, Owner: ownerAddress}
			err := clientNode.StoreProductString(productName, product)
			if err != nil {
				return
			}
		} else if action == "readproduct" {
			productName := params[1]
			product, ok := clientNode.ReadProductString(productName)
			if ok {
				fmt.Printf("the owner of the product: %s\n", product.Owner)
			} else {
				fmt.Println("this product doesn't exist. ")
			}
		} else {
			fmt.Println("unsupported action.")
		}
	}

	//clientNode.AddPeers("127.0.0.1:0")
	//time.Sleep(time.Second * 5)
	//
	//time.Sleep(time.Second * 1000)
	clientNode.Stop()
}
