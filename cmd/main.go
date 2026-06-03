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
		return initApp(args[1], defaultDBFile, defaultWalletFile)
	case "create-wallet":
		if len(args) != 2 {
			return fmt.Errorf("usage: create-wallet <username>[role]")
		}
		return createWallet(args[1], args[2], defaultWalletFile)
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
		return printChain(defaultDBFile, defaultWalletFile)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func PrintchainInfo(chainInfo *service.ChainInfo) error {
	blocks := chainInfo.Blocks
	blockHeight := chainInfo.Height
	fmt.Println()
	fmt.Printf("+++++++++ 区块链打印开始，当前区块高度 %d ++++++++++\n", blockHeight)
	for i, block := range blocks {
		fmt.Printf("========= 区块 %d =========\n", len(blocks)-i-1)
		fmt.Printf("当前区块哈希: %x\n", block.Hash)
		fmt.Printf("上一个区块哈希: %x\n", block.PrevHash)
		fmt.Println("++++++++++ 链结束符 ++++++++++")
		fmt.Println()
	}
	return nil
}

func initApp(minerAddress, dbFile string, dbWalletFile string) error {
	server, err := service.InitApp(minerAddress, dbFile, dbWalletFile)
	if err != nil {
		return fmt.Errorf("init chain: open services: %w", err)
	}
	defer server.Close()

	chainInfo, err := server.ChainService.RequireChainInfo()

	if server != nil {
		fmt.Println("node and chain initialized")
		PrintchainInfo(chainInfo)

	}

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

func createWallet(username, role, dbFile string) error {
	ws, closeFn, err := openWalletService(dbFile)
	if err != nil {
		return err
	}
	defer closeFn()

	err = ws.CreateWallet(username, role)
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
	ws, closeFn, err := openWalletService(dbFile)
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
	ws, closeFn, err := openWalletService(dbFile)
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
	ws, closeFn, err := openWalletService(dbFile)
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
	ws, closeFn, err := openWalletService(dbFile)
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

func printChain(chaindbFile string, walletdbFile string) error {
	appserver, err := service.OpenServices(chaindbFile, walletdbFile)
	if err != nil {
		return fmt.Errorf("open appserver: %w", err)
	}
	defer appserver.Close()

	chainInfo, err := appserver.ChainService.RequireChainInfo()
	if err != nil {
		return fmt.Errorf("print chain: require blockchain info: %w", err)
	}
	return PrintchainInfo(chainInfo)
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
