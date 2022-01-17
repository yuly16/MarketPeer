# MarketPeer


## Background of Chord

Chord is a peer-to-peer distributed hash table. Compared with a traditional central storage system, data is distributed evenly in every node in Chord. It decreases the burden of the central server because all nodes in Chord share the responsibility to store data. Chord describes how a key-value pair is stored in the node, and how to find the storage location of a key-value pair.

All nodes in Chord form a ring, and their locations are calculated by hashing their IP addresses. When storing a key-value pair, the component of Chord looks up which node is responsible for the key-value pair. Chord also supports the condition that a node joins the system and a node suddenly leaves the system. In our project, for simplicity, we don't consider the situation that a node fails or a node leaves the system. 

One advantage of Chord is that the node in the system doesn't have to be aware of all nodes. Each node only needs to know a small amount of routing information of other nodes. Specifically, each node only saves the information of its predecessor and successor. And it keeps the finger table. If a key is beyond its responsibility, it will search the finger table and send the key to other nodes. When the number of nodes grows exponentially, the search time of the location of the key grows linearly. 

 



## Overall System Architecture & Functionalities

![Screen Shot 2022-01-17 at 7.44.26 PM](arch.png)

- Blockchain stores the digial assets and digital currencies; Regulate the transaction exeution and verification; a "ledger"
- DHT stores all other metadatas. Like the mapping between network address and account address, owner list of each product, product list of each owner, etc.

### Messager

The Messager interface is adapted from homework implemented Peer. Yet, we made several modifications to make it easier to use in our case:

- send functionalities directly use `types.Message`, such that caller need not marshal themselves
- add `Registry` functionalities

```go
type Messager interface {
  peer.Service
	RegisterMessageCallback(types.Message, registry.Exec)
	Unicast(dest string, msg types.Message) error
	Broadcast(msg types.Message) error
	AddPeer(addr ...string)
	GetNeighbors() []string
}
```

Putting together, anything built on top of Messager only need to define their specific sending logic and register the callbacks. The Messager interface hides the complexity while not hide the power.

## BlockChain Architecture & Functionalities

### Overview

The two key abstractions are *Wallet* and *Miner*. Together, they form a FullNode in the blockchain network. They both rely on *Messager* to send, register and process the messages. The Wallet abstraction offers an entry point to the blockchain network. Our client use wallet provided APIs to interact with blockchain. The Miners are the main components of the blockchain network. They are responsible for storing the blocks, verifying the transactions and blocks and produce new blocks.

### Wallet

Wallet serves as the entry point to the blockchain network. Clients uses its provided APIs to interact with blockchain.

```go
type Wallet interface {
  SyncAccount() error
  ShowAccount() AccountInfo
  TransferToken(dest account.Address, amount int) error
  ProposeContract(code string) (string, error)
  TriggerContract(dest account.Address) error
}

```

- `SyncAccount` acts like a *refresh*, which will fetch the latest account state from the blockchain. `ShowAccount` just shows the synced Account information, like digital balance and digital assets.
- `TransferToken` will create a new transaction and submit it to the network. We note it is a blocking function. Not until the wallet is sure the transaction is solidly stored in the blockchain or a timeout occurs, the function will not return.
- `ProposeContract`'s overall procedure is similar to `TransferToken`. It issues a `CREATE_CONTRACT` transaction with supplied code. Then, it tries to make sure the transaction is stored until timeout occurs. Finally it returns the contract address and result.
- `TriggerContract` 's overall procedure is similar to `TransferToken`. The difference is that its dest is a contract address.

To faciliate above functionalites, we define several new messages: (1) `TransactionMessage` (2) `VerifyTransactionMessage` and `VerifyTransactionReplyMessage` (3) `SyncAccountMessage` and `SyncAccountReplyMessage`, where the reply messages is sent from the miners in the network.

### Miner

The miners are the key components of the blockchain network, they are responsible for 

- physically store the blocks in the blockchain
- verify and execute the transactions sent by Wallets
- produce new blocks based on Bitcoin-style PoW, and broadcast it to other miners
- verify the blocks broadcasted by other miners

As we adopt ethereum-alike blockchain network. The biggest difference from Bitcoin block is that it also contains the `WorldState`, which is a mapping between `AccountAddress` and `AccountState`. The `AccountState` is also a KV representing the digial assets. Further, we make serveral simplified assumptions: (1) the difficulty of the network is fixed, not dynamically adapted; (2) use hash map rather than Merkle tree as the KV interface. To faciliate above functionalites, we also define a new message: `BlockMessage` which is used to broadcast the block to other miners.

## Chord Architecture & Functionalities

### Overview

The chord system offers two APIs: Put(save a key-value pair into a node) and Get(find the owner of a key and read the value). Each node has an address in the chord. Usually, the address is the hash value of the IP address of the node. The following picture is an example of the chord system. All nodes form a ring, and key-value pairs are evenly distributed in this ring. 
![Screen Shot 2022-01-17 at 7.44.26 PM](chordArch.png)

### Predecessor and Successor
Each node saves the address of its predecessor and successor. In this example, the predecessor of N4 is N60, and the successor of N4 is N8. Each node should update its predecessor and successor frequently because a new node can join the system at any time. 

### Finger Table
Each node keeps its finger table locally. The finger table is used for finding the closest predecessor of a given key. For example, the following table shows the finger table of N4:

| distance | item |
| -------- | ---- |
| 1        | N8   |
| 2        | N8   |
| 4        | N8   |
| 8        | N18  |
| 16       | N30  |
| 32       | N43  |

When distance is 1, the id of the corresponding key is 4 + 1 = 5. The successor of this key is N8, so the item is N8. In the following section, we will discuss that the finger table is very useful to find the successor of a corresponding key.

### Find Successor
When we save a key into Chord, we would like to find the successor of this key. However, the node in Chord doesn't have a global view of the system, a node can't directly find the successor of this key. Therefore, a node searches the finger table and looks for the closest predecessor of this key, and then asks it to find the successor of this key.

### Join a new node
When a new node joins the Chord system, assume its hash value is k, firstly it should find its successor k* in the Chord. Secondly, it invokes Notify() to tell its successor that it has joined the system. When the successor receives the notify message, it sets the predecessor to ask and transfer some key-value pairs whose key is smaller than the new predecessor. 

### Updating the information of Chord
Each node should periodically invoke Notify() to tell the successor its existence. When the successor receives the notify, it judges whether the value of the node is larger than its previous predecessor; if yes, it sets a new predecessor. What's more, each node updates its finger table periodically. When a new node joins the system, Notify() only makes its predecessor and successor aware of its existence. It is the finger table that makes the whole system know that a new node joins. 


## Fine-grained team contribution

Main contribution is highlighted with bold text.

- Liangyong Yu:
  - Distributed Hash Table: **design, implement and test the Distributed Hash Table**
  - BlockChain: implement the execution of contract-creation 
  - SmartContract: None
  - Client: **design, implement and test the ClientNode in the market place**
  - CLI: **design and implement the CLI.**
- Lin Yuan:
  - Distributed Hash Table: None
  - BlockChain: implement the contract-related transaction submission, verification and execution.
  - SmartContract: **design, implement and test the SmartContract**
  - Client: None
  - CLI: starting code
- Yifei Li:
  - Distributed Hash Table: None
  - BlockChain: **design, implement and test the Blockchain functionalities.** 
  - SmartContract: None
  - Client: None
  - CLI: starting code and configuaration specifications.

## System Evaluation


### Chord evaluation

We create some unit tests to evaluate the correctness of Chord. For simplicity, all nodes wouldn't be failed. Our system should initialize a Chord system, whose nodes have the correct predecessor, successor, and finger table. What's more, When a new node joins the system, the whole system should be aware of its existence, and some keys stored in Chord should be transferred to the new node. 

The first unit test we build is to create a ten-node chord system. In the test program, we collect the hash value of each node and reconstruct the expected predecessors, successors, and finger tables of each node. And then we start the chord system. When the network is stable, we compare the predecessors, successors, and finger tables of each node and expected values. 

The second unit test is to store some key-value pairs and read them in Chord. First, we create a Chord system composed of ten nodes. When the network is stable, we generate some key-value pairs and store them in Chord, and then we read these key-values pairs from Chord. We evaluate that the owner of the key is correct, and values should be unchanged after saving and reading operations. 

The third unit test is to join some new nodes into Chord. First, we initialize a Chord system with only two nodes and store some key-value pairs into it. When the network is stable, eight nodes join the system simultaneously. In this process, the successors, predecessors, and finger tables of the original two nodes would change, and they should transfer some key-value pairs to the newly joined nodes. Finally, We read all key-value pairs from Chord and evaluate that the owner of the key is correct.


### Client evaluation
We warp Chord and Blockchain into a Client node. In this section, we create some unit tests to form the Chord system, transfer account, and execute transactions.

The first unit test is to from the Chord system. Like Chord evaluation, we create ten client nodes and form a Chord system, and insert some key-values pairs. We test that the network topology is correct, and key-value pairs are distributed into the correct client node.

The second unit test is to submit some transactions. We create three nodes. Each node submits a transaction. We evaluate that the blockchain in each node shows the transactions are saved in the block and the accounts of nodes change expectedly. 


The third unit test is to transfer accounts among client nodes. We create three nodes and ask node 1 to transfer some money to node 2. We evaluate that the transaction should be received by all nodes and be packed inside a block. 


### BlockChain Evaluation

- all good nodes
  - 3 nodes. node submit txns
  - what if fork happens
  - concurrent submit txns

- attackers to the PoW
  - fork handles
  - attacker cannot rule the network. as long as it is below 51%
  - 51% attack

- - 







## unused

