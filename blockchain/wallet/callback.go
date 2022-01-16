package wallet

//func (w *Wallet) WalletTxnMsgCallback(msg types.Message, pkt transport.Packet) error {
//	signedTxnMsg := msg.(*types.WalletTransactionMessage)
//	signedTxn := signedTxnMsg.Txn
//	fmt.Println(w.addr)
//	publicKey, err := crypto.Ecrecover(signedTxn.Digest, signedTxn.Signature)
//	if err != nil {
//		return err
//	}
//	okValidSignature := crypto.VerifySignature(publicKey, signedTxn.Digest,
//		signedTxn.Signature[:len(signedTxn.Signature)-1])
//
//	okValidPublickey := signedTxn.Txn.From == *account.NewAddressFromPublicKey(publicKey)
//	fmt.Printf("I am %s, the validation of the transaction sent from %s is %t\n",
//		w.addr, pkt.Header.Source, okValidSignature && okValidPublickey)
//	return nil
//}
