package main

import "fmt"

func CreatWalletParsing(args []string) (username string, role string, err error) {
	if len(args) < 2 || len(args) > 3 {
		return "", "", fmt.Errorf("too many or too less arguments")
	}
	if len(args) == 2 {
		username = args[1]
		role = ""
	}

	if len(args) == 3 {
		username = args[1]
		role = args[2]
	}

	return username, role, nil

}

func InitParsing(args []string) (minerAddress string, err error) {
	if len(args) < 1 || len(args) > 2 {
		return "", fmt.Errorf("too many or too less arguments")
	}

	if len(args) == 1 {
		minerAddress = ""
	}

	if len(args) == 2 {
		minerAddress = args[1]
	}

	return minerAddress, nil
}
