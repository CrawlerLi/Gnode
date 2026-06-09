package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/CrawlerLi/myMiniBitcoin/internal/config"
	"github.com/CrawlerLi/myMiniBitcoin/internal/infra/database"
	"github.com/CrawlerLi/myMiniBitcoin/internal/node"
	"github.com/CrawlerLi/myMiniBitcoin/internal/service"
	"github.com/CrawlerLi/myMiniBitcoin/internal/wallet"
)

const defaultDBFile = "cmd/blockchain.db"
const defaultWalletFile = "cmd/wallet.db"
const defaultConfigFile = "./configs/default_node.json"

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
		minerAdress, configFilePath, err := InitParsing(args)
		if err != nil {
			return fmt.Errorf("%w :usage: init <minerAddress>", err)
		}
		return initApp(minerAdress, configFilePath)
	case "create-wallet":
		username, role, err := CreatWalletParsing(args)
		if err != nil {
			return fmt.Errorf("%w: usage: create-wallet <username>[role(optional)]", err)
		}
		return createWallet(username, role, defaultWalletFile)
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
	case "reset-chain":
		return resetChain(args, defaultDBFile, defaultWalletFile)
	case "node":
		configFilepath, err := NodeParsing(args)
		if err != nil {
			return fmt.Errorf("%w: usage: node <config file path>", err)
		}
		return runNode(configFilepath)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func PrintchainInfo(chainInfo *service.ChainInfo) error {
	blocks := chainInfo.Blocks
	blockHeight := chainInfo.Height
	fmt.Println()
	fmt.Printf("+++++++++ START PRINTING BLOCKCHAIN, CURRENT BLOCK HEIGHT IS %d ++++++++++\n", blockHeight)
	for i, block := range blocks {
		fmt.Printf("========= BLOCKCHAIN %d =========\n", len(blocks)-i-1)
		fmt.Printf("CURRENT BLOCKCHAIN HASH: %x\n", block.Hash)
		fmt.Printf("PREVIOUS BLOCKCHAIN HASH: %x\n", block.PrevHash)
		fmt.Println()
	}
	fmt.Println("++++++++++++++++++ BLOCKCHAIN PRINTING ENDING SYMBOL  ++++++++++++++++++++")
	return nil
}

func initApp(minerAddress, configFilePath string) error {
	nodeConfig, err := config.Load(configFilePath)
	if err != nil {
		return fmt.Errorf("init app: load init config file: %w", err)
	}
	chainDBFile := nodeConfig.ChainDB
	dbWalletFile := nodeConfig.WalletDB
	if err != nil {
		return fmt.Errorf("init app: %w", err)
	}
	initialized, err := service.IsChainInitialized(chainDBFile)
	if err != nil {
		return fmt.Errorf("init app: check chain initialized: %w", err)
	}
	if initialized {
		return fmt.Errorf("init app: chain already initialized")
	}
	server, err := service.InitApp(minerAddress, chainDBFile, dbWalletFile)
	if err != nil {
		return fmt.Errorf("init chain: open services: %w", err)
	}
	defer server.Close()

	chainInfo, err := server.ChainService.RequireChainInfo()

	if server != nil {
		fmt.Println("node and chain initialized")
		PrintchainInfo(chainInfo)
	}

	if err != nil {
		return fmt.Errorf("init chain: require blockchain status after initialization: %w", err)
	}

	return nil
}

func openWalletService(walletDBFile string) (*service.WalletService, func() error, error) {
	db, err := database.OpenDB(walletDBFile)
	if err != nil {
		return nil, nil, fmt.Errorf("open wallet db: %w", err)
	}

	if err := db.CreateBucket("Wallet"); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("create wallet bucket: %w", err)
	}

	ws := service.NewWalletService(wallet.NewWalletStorage(db), nil)
	return ws, db.Close, nil
}

func openAppWalletService(chainDBFile string, walletDBFile string) (*service.WalletService, func() error, error) {
	app, err := service.OpenServices(chainDBFile, walletDBFile)
	if err != nil {
		return nil, nil, err
	}

	return app.WalletService, app.Close, nil
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

	fmt.Printf("wallet created: user=%s address=%s, role=%s\n", info.Username, info.Address, info.Role)
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
		fmt.Printf("user=%s address=%s role=%s\n", item.Username, item.Address, item.Role)
	}
	return nil
}

func getBalance(username, dbFile string) error {
	ws, closeFn, err := openAppWalletService(defaultDBFile, dbFile)
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
	ws, closeFn, err := openAppWalletService(defaultDBFile, dbFile)
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

func resetChain(args []string, chainDBFile string, walletDBFile string) error {
	if len(args) > 2 {
		return fmt.Errorf("usage: reset-chain [--with-wallets]")
	}

	withWallets := false
	if len(args) == 2 {
		if args[1] != "--with-wallets" {
			return fmt.Errorf("usage: reset-chain [--with-wallets]")
		}
		withWallets = true
	}

	if err := removeDBFile(chainDBFile); err != nil {
		return fmt.Errorf("reset chain: %w", err)
	}
	fmt.Printf("removed chain database: %s\n", chainDBFile)

	if withWallets {
		if err := removeDBFile(walletDBFile); err != nil {
			return fmt.Errorf("reset wallets: %w", err)
		}
		fmt.Printf("removed wallet database: %s\n", walletDBFile)
	}

	return nil
}

func removeDBFile(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("remove %s: %w", path, err)
}

func runNode(configFilePath string) error {
	nodeConfig, err := config.Load(configFilePath)
	if err != nil {
		return fmt.Errorf("run Node: load config file: %w", err)
	}

	appService, err := service.OpenServices(nodeConfig.ChainDB, nodeConfig.WalletDB)
	if err != nil {
		return fmt.Errorf("run node: %w", err)
	}
	defer appService.Close()

	n, err := node.InitNode(appService, nodeConfig.NodeID, nodeConfig.ListenAddr, nodeConfig.Peers)
	if err != nil {
		return fmt.Errorf("run node: %w", err)
	}

	defer n.Stop()

	n.Start()
	fmt.Printf("node %s listening on %s\n", n.ID, n.Addr)
	for peerAddr := range n.Peers {
		fmt.Printf("connected peer: %s\n", peerAddr)
		//for ping test
		resp, err := n.PingPeer(peerAddr)
		if err != nil {
			return fmt.Errorf("failed to ping peer %s: %w", peerAddr, err)
		}
		fmt.Printf("Received ping response [%s] from %s", resp, peerAddr)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	select {
	case <-interrupt:
		fmt.Println("node shutting down")
		return nil
	case err := <-n.Errch():
		if err != nil {
			return fmt.Errorf("run node: node stopped unexpectedly: %w", err)
		}
	}
	return nil
}

func printUsage() {
	fmt.Println("myMiniBitcoin CLI")
	fmt.Println()
	fmt.Println("usage:")
	fmt.Println("  go run ./cmd <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  init  [configFilePath]  [minerAddress]      initialize chain and genesis block")
	fmt.Println("  create-wallet <username> [role]            create and persist a wallet, role can be 'user' or 'miner', default is 'user'")
	fmt.Println("  get-wallet <username>                      show wallet detail")
	fmt.Println("  list-wallets                               list all wallets")
	fmt.Println("  balance <username>                    	  query wallet balance")
	fmt.Println("  transfer <fromUser> <toAddress> <amount>   transfer coin and mine one block")
	fmt.Println("  print                                      print blockchain")
	fmt.Println("  reset-chain [--with-wallets]               remove local chain database, optionally remove wallets")
	fmt.Println("  node <config file path>                    run node and network")
}
