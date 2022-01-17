# MarketPeer: A P2P Digital Market Platform

Team: Yifei Li, Liangyong Yu, Lin Yuan

Nowadays, e-commerce has become an indispensable part of our lives. Despite the convenience of these centralized platforms, they have several drawbacks. First of all, sellers and buyers each need to pay fees for the platform. Secondly, although maintained and secured by world's best engineers, these platforms are still under the risk of malicious attacks. We propose to establish a peer-to-peer online shopping platform, where the trades directly happen between buyers and sellers. Smart contract can make the transaction independently and automatically without the need of third-party organizations. In this "MarketPeer" platform, peers not only get rid of platform fees but also obtain better security.

## How-to start MarketPeer

We have created a configuration file in `./cli/config.json`. Users can also modify it to configure its own network. Run command `go run main.go <IP address>` to initiate the CLI of node. There are 3 nodes in the configured network. Then you can follow the instructions in CLI to trade in MarketPeer.

```
:~/MarketPeer$  cd cli
:~/MarketPeer/cli$ go run main.go 127.0.0.1:51000
:~/MarketPeer/cli$ go run main.go 127.0.0.1:52000
:~/MarketPeer/cli$ go run main.go 127.0.0.1:53000
```

