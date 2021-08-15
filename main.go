package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/WithGJR/regit-go/core"
)

func main() {
	commitCmd := flag.NewFlagSet("commit", flag.ExitOnError)
	var commitMessage string
	commitCmd.StringVar(&commitMessage, "m", "", "A commmit message")

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("command too short")
		os.Exit(1)
	}

	regit := core.NewReGit(workingDir)

	switch os.Args[1] {
	case "init":
		regit.Init()
	case "add":
		if len(os.Args) == 2 {
			fmt.Println("Nothing specified, nothing added.")
			os.Exit(1)
		}
		regit.Add(os.Args[2:])
	case "commit":
		commitCmd.Parse(os.Args[2:])
		if commitMessage == "" {
			fmt.Println("-m option is required.")
			os.Exit(1)
		}
		regit.Commmit(commitMessage)
	case "checkout":
		if len(os.Args) == 2 {
			fmt.Println("Error: you need to specify path names")
			os.Exit(1)
		}
		regit.Checkout(os.Args[2:])
	default:
		fmt.Println("'" + os.Args[1] + "' is not a ReGit command.")
	}
}
