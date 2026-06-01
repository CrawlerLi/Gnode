package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/CrawlerLi/myMiniBitcoin/internal/core"
	"github.com/CrawlerLi/myMiniBitcoin/internal/service"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
	"github.com/CrawlerLi/myMiniBitcoin/pkg/crypto"
)

const defaultDBFile = "cmd/mini_bitcoin.db"
const defaultWalletFile = "cmd/mini_bitcoin_wallet.dat"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		runInteractive()
		return
	}

	if err := run(args); err != nil {
		log.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

func runInteractive() {
	printUsage()
	fmt.Println(`type "help" for commands, "exit" to quit`)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println()
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return
		}

		args := strings.Fields(line)
		if err := run(args); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "init":
		if len(args) != 2 {
			return fmt.Errorf("usage: init <minerAddress>")
		}
		return initChain(args[1], defaultDBFile)
	case "create-wallet":
		if len(args) != 2 {
			return fmt.Errorf("usage: create-wallet <username>")
		}
		return createWallet(args[1], defaultWalletFile)
	case "get-wallet":
		if len(args) != 2 {
			return fmt.Errorf("usage: get-wallet <username>")
		}
		return getWallet(args[1], defaultWalletFile)
	case "list-wallets":
		if len(args) != 1 {
			return fmt.Errorf("usage: list-wallets")
		}
		return listWallets(defaultWalletFile)
	case "balance":
		if len(args) != 2 {
			return fmt.Errorf("usage: balance <username>")
		}
		return getBalance(args[1], defaultWalletFile)
	case "transfer":
		if len(args) != 4 {
			return fmt.Errorf("usage: transfer <fromUser> <toAddress> <amount>")
		}
		amount, err := strconv.Atoi(args[3])
		if err != nil {
			return fmt.Errorf("invalid amount %q: %w", args[3], err)
		}
		return transfer(args[1], args[2], amount, defaultWalletFile)
	case "print":
		if len(args) != 1 {
			return fmt.Errorf("usage: print")
		}
		return printChain(defaultDBFile)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func initChain(minerAddress, dbFile string) error {
	minerPubKeyHash, err := crypto.AddressToPubkeyHash([]byte(minerAddress))
	if err != nil {
		return fmt.Errorf("init chain: parse miner address: %w", err)
	}

	bc, err := core.InitBlockChain(minerPubKeyHash, dbFile)
	if err != nil {
		return fmt.Errorf("init chain: %w", err)
	}
	defer bc.DB.Close()

	err = bc.DB.CreateBucket("Wallet")
	if err != nil {
		return fmt.Errorf("init chain: create wallet bucket: %w", err)
	}

	fmt.Println("blockchain initialized")
	fmt.Printf("db: %s\n", dbFile)
	fmt.Printf("genesis reward to: %s\n", minerAddress)
	return nil
}

func openWalletService(dbFile string) (*service.WalletService, func() error, error) {
	bc, err := core.OpenBlockChain(dbFile)
	if err != nil {
		return nil, nil, fmt.Errorf("open blockchain: %w", err)
	}

	ws := service.NewWalletService(wallet.NewWalletStorage(bc.DB), bc)
	return ws, bc.DB.Close, nil
}

func createWallet(username, dbFile string) error {
	ws, closeFn, err := openWalletService(dbFile)
	if err != nil {
		return err
	}
	defer closeFn()

	err = ws.CreateWallet(username)
	if err != nil {
		return fmt.Errorf("create wallet: %w", err)
	}

	info, err := ws.GetWallet(username)
	if err != nil {
		return fmt.Errorf("create wallet: read back wallet: %w", err)
	}

	fmt.Printf("wallet created: user=%s address=%s\n", info.Username, info.Address)
	return nil
}

func getWallet(username, dbFile string) error {
	ws, closeFn, err := openWalletServer(dbFile)
	if err != nil {
		return err
	}
	defer closeFn()

	info, err := ws.GetWallet(username)
	if err != nil {
		return fmt.Errorf("get wallet: %w", err)
	}

	fmt.Printf("username: %s\n", info.Username)
	fmt.Printf("address : %s\n", info.Address)
	fmt.Printf("pubkey  : %s\n", info.PublicKey)
	return nil
}

func listWallets(dbFile string) error {
	ws, closeFn, err := openWalletServer(dbFile)
	if err != nil {
		return err
	}
	defer closeFn()

	wallets, err := ws.ListWallets()
	if err != nil {
		return fmt.Errorf("list wallets: %w", err)
	}

	if len(wallets) == 0 {
		fmt.Println("no wallets")
		return nil
	}

	for _, item := range wallets {
		fmt.Printf("user=%s address=%s\n", item.Username, item.Address)
	}
	return nil
}

func getBalance(username, dbFile string) error {
	ws, closeFn, err := openWalletServer(dbFile)
	if err != nil {
		return err
	}
	defer closeFn()

	balance, err := ws.GetBalance(username)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	fmt.Printf("%s balance: %d\n", username, balance)
	return nil
}

func transfer(fromUser, toAddress string, amount int, dbFile string) error {
	ws, closeFn, err := openWalletServer(dbFile)
	if err != nil {
		return err
	}
	defer closeFn()

	res, err := ws.Transfer(fromUser, toAddress, amount)
	if err != nil {
		return fmt.Errorf("transfer: %w", err)
	}

	fmt.Printf("transfer success, txid=%s\n", res.TxID)
	return nil
}

func printChain(dbFile string) error {
	bc, err := core.OpenBlockChain(dbFile)
	if err != nil {
		return fmt.Errorf("open blockchain: %w", err)
	}
	defer bc.DB.Close()

	return bc.Print()
}

func printUsage() {
	fmt.Println("myMiniBitcoin CLI")
	fmt.Println()
	fmt.Println("usage:")
	fmt.Println("  go run ./cmd <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  init [minerAddress]                   initialize chain and genesis block")
	fmt.Println("  create-wallet <username> [role]       create and persist a wallet, role can be 'user' or 'miner', default is 'user'")
	fmt.Println("  get-wallet <username>                 show wallet detail")
	fmt.Println("  list-wallets                          list all wallets")
	fmt.Println("  balance <username>                    query wallet balance")
	fmt.Println("  transfer <fromUser> <toAddress> <n>   transfer coin and mine one block")
	fmt.Println("  print                                 print blockchain")
}
