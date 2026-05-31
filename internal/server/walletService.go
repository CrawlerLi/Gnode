package server

import (
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

type WalletServer struct {
	store *wallet.WalletStorage
	bc    *core.BlockChain
}

type WalletInfo struct {
	Username  string
	Address   string
	PublicKey string // optional: hex
}

type TransferReslut struct {
	txID string
}

const (
	addressVersionLen  = 1
	addressChecksumLen = 4
	pubKeyHashLen      = 20
)

func NewWalletServer(store *wallet.WalletStorage, bc *core.BlockChain) *WalletServer {
	return &WalletServer{store: store, bc: bc}
}

func (ws *WalletServer) CreateWallet(username string) error {
	newWallet, err := wallet.NewWallet()
	if err != nil {
		return fmt.Errorf("ceate wallet: generate wallet in memory: %w", err)
	}

	err = ws.store.Save(username, newWallet)
	if err != nil {
		return fmt.Errorf("create wallet: save wallet: %w", err)
	}

	return nil
}

func (ws *WalletServer) GetWallet(username string) (*WalletInfo, error) {
	wallet, err := ws.store.Get(username)
	if err != nil {
		return nil, fmt.Errorf("get wallet info : %w", err)
	}

	wInfo := &WalletInfo{
		Username:  username,
		Address:   string(wallet.Address),
		PublicKey: fmt.Sprintf("%x", wallet.Publickey),
	}

	return wInfo, nil

}

func (ws *WalletServer) ListWallets() ([]*WalletInfo, error) {
	walletsList, err := ws.store.List()
	if err != nil {
		return nil, fmt.Errorf("List wallets: get wallets list: %w", err)
	}

	var walletInfolist []*WalletInfo
	for username, wallet := range walletsList {
		wInfo := &WalletInfo{
			Username:  username,
			Address:   string(wallet.Address),
			PublicKey: fmt.Sprintf("%x", wallet.Publickey),
		}
		walletInfolist = append(walletInfolist, wInfo)
	}
	return walletInfolist, nil
}

func (ws *WalletServer) DetelteWallet(username string) error {
	err := ws.store.Delete(username)
	if err != nil {
		return err
	}
	return nil
}

func (ws *WalletServer) GetBalance(username string) (int, error) {
	wallet, err := ws.store.Get(username)
	if err != nil {
		return 0, fmt.Errorf("get banlance: get wallet in database: %w", err)
	}

	var balance int
	pubkeyHash, err := crypto.AddressToPubkeyHash([]byte(wallet.Address))
	if err != nil {
		return 0, fmt.Errorf("get balance: convert address to pubkey hash: %w", err)
	}

	utxos, err := ws.bc.UTXO.Snapshot()
	if err != nil {
		return 0, fmt.Errorf("fail to snapshot UTXO: %w", err)
	}
	for _, utxo := range utxos {
		if string(utxo.ScriptPubkey) == string(pubkeyHash) {
			balance += utxo.Value
		}
	}

	return balance, nil
}

func (ws *WalletServer) Transfer(fromUser string, to string, amount int, bc *core.BlockChain) (*TransferReslut, error) {
	if fromUser == "" || to == "" || amount <= 0 {
		return nil, fmt.Errorf("invalid transfer params")
	}

	wallet, err := ws.store.Get(fromUser)
	if err != nil {
		return nil, fmt.Errorf("Transfer coin: get send wallet : %w", err)
	}

	var tx *core.Transaction
	pubkeyHash := crypto.HashPubkey(wallet.Publickey)

	payable, acc, err := bc.UTXO.FindSpendableUTXOS(amount, pubkeyHash)
	if err != nil {
		return nil, fmt.Errorf("Failed to find payable UTXO: %w", err)
	}

	if acc < amount {
		return nil, fmt.Errorf("banlance do not enough")
	}

	var Vin []core.TxInput
	prevOutputs := make(map[string]core.TxOutput)

	for _, spendableUTXO := range payable {

		txID, idx := spendableUTXO.OutPoint.TxID, spendableUTXO.OutPoint.OutIndex
		txin := core.TxInput{
			Txid:     txID,
			OutIndex: idx,
			Pubkey:   wallet.Publickey,
		}

		Vin = append(Vin, txin)
		prevOutputs[spendableUTXO.OutPoint.String()] = spendableUTXO.Output

	}

	TopubkeyHash, err := crypto.AddressToPubkeyHash([]byte(to))
	if err != nil {
		return nil, fmt.Errorf("create new tx: convert recipient address to pubkey hash: %w", err)
	}
	if len(TopubkeyHash) != pubKeyHashLen {
		return nil, fmt.Errorf("create new tx: invalid recipient pubkey hash length")
	}

	txout := core.TxOutput{
		Value:        amount,
		ScriptPubkey: TopubkeyHash,
	}

	Vout := []core.TxOutput{txout}

	if acc > amount {
		Vout = append(Vout, core.TxOutput{Value: acc - amount, ScriptPubkey: pubkeyHash})
	}

	tx = &core.Transaction{
		Vin:  Vin,
		Vout: Vout,
	}

	txID, err := tx.Hash()
	if err != nil {
		return nil, fmt.Errorf("create new tx: hash new tx: %w", err)
	}
	tx.ID = txID

	err = wallet.Sign(tx, prevOutputs)
	if err != nil {
		return nil, fmt.Errorf("create new tx: sign new tx: %w", err)
	}

	NewCoinbaseTx, err := core.NewCoinBase(crypto.HashPubkey(wallet.Publickey))
	if err != nil {
		return nil, fmt.Errorf("Transfer: create new coinbase tx: %w", err)
	}

	// A memory pool will be implemented here later
	err = ws.bc.AddBlock([]*core.Transaction{tx, NewCoinbaseTx})
	if err != nil {
		return nil, fmt.Errorf("Transfer: write data on-chian %w", err)
	}

	return &TransferReslut{txID: fmt.Sprintf("%x", tx.ID)}, nil
}
