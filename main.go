package main

import "fmt"

func main() {

	Alice := NewWallet()
	Bob := NewWallet()

	fmt.Println("The wallets have been constructed Successfully")
	fmt.Println("The address of Alice is ", string(Alice.Address))
	fmt.Println("The address of Bob is ", string(Bob.Address))
	fmt.Println()

	bc := NewBlockChain(HashPubkey(Alice.Publickey))
	fmt.Println("The gensis block has been created!")
	bc.Print()

	bc.AddBlock([]*Transaction{NewCoinBase(HashPubkey(Alice.Publickey))})
	fmt.Println("The second block has been created")
	bc.Print()

	fmt.Println("======初始余额======")
	banlanceA := bc.GetBalance(string(Alice.Address))
	fmt.Println("The banlance of Alice is ", banlanceA)

	banlanceB := bc.GetBalance(string(Bob.Address))
	fmt.Println("The banlance of Bob is ", banlanceB)
	fmt.Println()

	// 打包一笔交易（这里先用 coinbase 模拟）
	bc.AddBlock([]*Transaction{NewTrasaction(Alice, string(Bob.Address), 30, bc),
		NewCoinBase(HashPubkey(Alice.Publickey)),
	})
	fmt.Println("The third block has been created, all transacrions have benn verified!")
	bc.Print()

	fmt.Println("======最终余额======")
	banlanceA = bc.GetBalance(string(Alice.Address))
	fmt.Println("The banlance of Alice is ", banlanceA)

	banlanceB = bc.GetBalance(string(Bob.Address))
	fmt.Println("The banlance of Bob is ", banlanceB)
	fmt.Println()

	fmt.Println("You are good man!")

}
