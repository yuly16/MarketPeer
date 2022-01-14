package storage

type MerkleTrie struct {
	KV
	root *MerkleTreeNode
}

type MerkleTreeNode struct {
}
