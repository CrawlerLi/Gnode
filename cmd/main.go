package main

import (
	"fmt"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

func main() {

	Alice := crypto.NewWallet()
	Bob := crypto.NewWallet()

	fmt.Println("The wallets have been constructed Successfully")
	fmt.Println("The address of Alice is ", string(Alice.Address))
	fmt.Println("The address of Bob is ", string(Bob.Address))
	fmt.Println()

	bc := core.NewBlockChain(crypto.HashPubkey(Alice.Publickey))
	fmt.Println("The gensis block has been created!")
	bc.Print()

	bc.AddBlock([]*core.Transaction{core.NewCoinBase(crypto.HashPubkey(Alice.Publickey))})
	fmt.Println("The second block has been created")
	bc.Print()

	fmt.Println("======初始余额======")
	banlanceA := wallet.GetBalance(bc, string(Alice.Address))
	fmt.Println("The banlance of Alice is ", banlanceA)

	banlanceB := wallet.GetBalance(bc, string(Bob.Address))
	fmt.Println("The banlance of Bob is ", banlanceB)
	fmt.Println()

	// 打包一笔交易
	bc.AddBlock([]*core.Transaction{wallet.NewTrasaction(Alice, string(Bob.Address), 30, bc),
		core.NewCoinBase(crypto.HashPubkey(Alice.Publickey)),
	})
	fmt.Println("The third block has been created, all transactions have benn verified!")
	bc.Print()

	fmt.Println("======最终余额======")
	banlanceA = wallet.GetBalance(bc, string(Alice.Address))
	fmt.Println("The banlance of Alice is ", banlanceA)

	banlanceB = wallet.GetBalance(bc, string(Bob.Address))
	fmt.Println("The banlance of Bob is ", banlanceB)
	fmt.Println()

	fmt.Println("You are good man!")

}
