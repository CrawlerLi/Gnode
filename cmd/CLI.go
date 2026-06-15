package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/CrawlerLi/Gnode/internal/config"
	"github.com/CrawlerLi/Gnode/internal/node"
	"github.com/CrawlerLi/Gnode/internal/service"
)

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
			return fmt.Errorf("%w: usage: init [configFilePath] [minerAddress]", err)
		}
		return initLocalApp(minerAdress, configFilePath)
	case "sync-init":
		minerAdress, configFilePath, err := InitParsing(args)
		if err != nil {
			return fmt.Errorf("%w: usage: init [configFilePath] [minerAddress]", err)
		}
		return syncInit(minerAdress, configFilePath)
	case "config":
		return runConfigCommand(args)
	case "create-wallet":
		username, role, err := CreatWalletParsing(args)
		if err != nil {
			return fmt.Errorf("%w: usage: create-wallet <username>[role(optional)]", err)
		}
		nodeConfig, err := loadNodeConfig("")
		if err != nil {
			return fmt.Errorf("create wallet: %w", err)
		}
		return createWallet(username, role, nodeConfig.WalletDB)
	case "get-wallet":
		if len(args) != 2 {
			return fmt.Errorf("usage: get-wallet <username>")
		}
		nodeConfig, err := loadNodeConfig("")
		if err != nil {
			return fmt.Errorf("get wallet: %w", err)
		}
		return getWallet(args[1], nodeConfig.WalletDB)
	case "list-wallets":
		if len(args) != 1 {
			return fmt.Errorf("usage: list-wallets")
		}
		nodeConfig, err := loadNodeConfig("")
		if err != nil {
			return fmt.Errorf("list wallets: %w", err)
		}
		return listWallets(nodeConfig.WalletDB)
	case "balance":
		if len(args) != 2 {
			return fmt.Errorf("usage: balance <username>")
		}
		nodeConfig, err := loadNodeConfig("")
		if err != nil {
			return fmt.Errorf("balance: %w", err)
		}
		return getBalance(args[1], nodeConfig.ChainDB, nodeConfig.WalletDB)
	case "transfer":
		if len(args) != 4 {
			return fmt.Errorf("usage: transfer <fromUser> <toAddress> <amount>")
		}
		amount, err := strconv.Atoi(args[3])
		if err != nil {
			return fmt.Errorf("invalid amount %q: %w", args[3], err)
		}
		nodeConfig, err := loadNodeConfig("")
		if err != nil {
			return fmt.Errorf("transfer: %w", err)
		}
		return transfer(args[1], args[2], amount, nodeConfig.ChainDB, nodeConfig.WalletDB)
	case "print":
		if len(args) != 1 {
			return fmt.Errorf("usage: print")
		}
		nodeConfig, err := loadNodeConfig("")
		if err != nil {
			return fmt.Errorf("print: %w", err)
		}
		return printChain(nodeConfig.ChainDB, nodeConfig.WalletDB)
	case "reset-chain":
		nodeConfig, err := loadNodeConfig("")
		if err != nil {
			return fmt.Errorf("reset chain: %w", err)
		}
		return resetChain(args, nodeConfig.ChainDB, nodeConfig.WalletDB)
	case "node":
		configFilepath, err := NodeParsing(args)
		if err != nil {
			return fmt.Errorf("%w: usage: node [configFilePath]", err)
		}
		return runNode(configFilepath)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func loadNodeConfig(configFilePath string) (*config.NodeConfig, error) {
	resolvedPath, err := resolveNodeConfigPath(configFilePath)
	if err != nil {
		return nil, err
	}

	nodeConfig, err := config.Load(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("load node config %s: %w", resolvedPath, err)
	}

	return nodeConfig, nil
}

func resolveNodeConfigPath(configFilePath string) (string, error) {
	if strings.TrimSpace(configFilePath) != "" {
		return configFilePath, nil
	}

	path, err := config.ActiveNodeConfigPath()
	if err != nil {
		return "", fmt.Errorf("load active node config path: %w", err)
	}

	return path, nil
}

func PrintchainInfo(chainInfo *service.ChainInfo) error {
	blocks := chainInfo.Blocks
	blockHeight := chainInfo.Height
	fmt.Println()
	fmt.Printf("+++++++++ START PRINTING BLOCKCHAIN, CURRENT BLOCK HEIGHT IS %d ++++++++++\n", blockHeight)
	for _, block := range blocks {
		fmt.Printf("========= BLOCKCHAIN %d =========\n", block.Height)
		fmt.Printf("CURRENT BLOCKCHAIN HASH: %x\n", block.Hash)
		fmt.Printf("PREVIOUS BLOCKCHAIN HASH: %x\n", block.PrevHash)
		fmt.Printf("Number of including transaction: %d\n", block.TxNums)

		for i, txInfo := range block.Txs {
			fmt.Printf("***Tx %d ***\n", i)
			fmt.Printf("The Tx id is %x\n", txInfo.Txid)
			fmt.Printf("The Nums of Vin is %d\n", txInfo.VinNums)
			fmt.Printf("The Nums of Vout is %d\n", txInfo.VoutNums)
		}
		fmt.Println()
	}
	fmt.Println("++++++++++++++++++ BLOCKCHAIN PRINTING ENDING SYMBOL  ++++++++++++++++++++")
	return nil
}

func initLocalApp(minerAddress, configFilePath string) error {
	nodeConfig, err := loadNodeConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("init local app: %w", err)
	}
	chainDBFile := nodeConfig.ChainDB
	dbWalletFile := nodeConfig.WalletDB
	initialized, err := service.IsChainInitialized(chainDBFile)
	if err != nil {
		return fmt.Errorf("init local app: check chain initialized: %w", err)
	}
	if initialized {
		return fmt.Errorf("init local app: chain already initialized")
	}
	server, err := service.InitLocalApp(minerAddress, chainDBFile, dbWalletFile)
	if err != nil {
		return fmt.Errorf("init local app: open services: %w", err)
	}
	defer server.Close()

	chainInfo, err := server.ChainService.GetChainInfo()
	if err != nil {
		return fmt.Errorf("init local app: require blockchain status after initialization: %w", err)
	}
	if server != nil {
		log.Println("node and chain initialized")
		PrintchainInfo(chainInfo)
	}

	return nil
}

func syncInit(minerAddress, configFilePath string) error {
	nodeConfig, err := loadNodeConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("sync init: %w", err)
	}
	chainDBFile := nodeConfig.ChainDB
	dbWalletFile := nodeConfig.WalletDB
	initialized, err := service.IsChainInitialized(chainDBFile)
	if err != nil {
		return fmt.Errorf("sync init: check chain initialized: %w", err)
	}
	if initialized {
		return fmt.Errorf("sync init: chain already initialized")
	}
	server, err := service.SyncInit(minerAddress, chainDBFile, dbWalletFile)
	if err != nil {
		return fmt.Errorf("init local app: open services: %w", err)
	}
	log.Printf("node and chain initialize for sync")
	defer server.Close()
	return nil
}

func createWallet(username, role, dbFile string) error {
	ws, closeFn, err := service.OpenWalletService(dbFile)
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
	ws, closeFn, err := service.OpenWalletService(dbFile)
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
	ws, closeFn, err := service.OpenWalletService(dbFile)
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

func getBalance(username, chainDBFile string, walletDBFile string) error {
	appService, err := service.OpenServices(chainDBFile, walletDBFile)
	if err != nil {
		return err
	}
	defer appService.Close()

	balance, err := appService.WalletService.GetBalance(username)
	if err != nil {
		return fmt.Errorf("get balance: %w", err)
	}

	fmt.Printf("%s balance: %d\n", username, balance)
	return nil
}

func transfer(fromUser, toAddress string, amount int, chainDBFile string, walletDBFile string) error {
	appService, err := service.OpenServices(chainDBFile, walletDBFile)
	if err != nil {
		return err
	}
	defer appService.Close()

	res, err := appService.WalletService.Transfer(fromUser, toAddress, amount)
	if err != nil {
		return fmt.Errorf("transfer: %w", err)
	}

	fmt.Printf("transfer success, txid=%s\n", res.TxID)
	return nil
}

func printChain(chaindbFile string, walletdbFile string) error {
	appserver, err := service.OpenServices(chaindbFile, walletdbFile)
	if err != nil {
		return fmt.Errorf("print chain: open appserver: %w", err)
	}
	defer appserver.Close()

	chainInfo, err := appserver.ChainService.GetChainInfo()
	if err != nil {
		return fmt.Errorf("print chain: %w", err)
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
	nodeConfig, err := loadNodeConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("run node: %w", err)
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

	err = n.Start()

	return err

}

func printUsage() {
	fmt.Println("GNODE CLI")
	fmt.Println()
	fmt.Println("usage:")
	fmt.Println("  go run ./cmd <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  init  [configFilePath] [minerAddress]      initialize chain and create genesis block")
	fmt.Println("  sync-init [configFilePath] [minerAddress]  initialize for sync from peer")
	fmt.Println("  config show                                show active node config path")
	fmt.Println("  config use <configFilePath>                set active node config path")
	fmt.Println("  config reset                               reset active node config to default")
	fmt.Println("  create-wallet <username> [role]            create and persist a wallet, role can be 'user' or 'miner', default is 'user'")
	fmt.Println("  get-wallet <username>                      show wallet detail")
	fmt.Println("  list-wallets                               list all wallets")
	fmt.Println("  balance <username>                    	  query wallet balance")
	fmt.Println("  transfer <fromUser> <toAddress> <amount>   transfer coin and mine one block")
	fmt.Println("  print                                      print blockchain")
	fmt.Println("  reset-chain [--with-wallets]               remove local chain database, optionally remove wallets")
	fmt.Println("  node [configFilePath]                      run node and network")
}
