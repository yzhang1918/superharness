package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/catu-ai/easyharness/internal/contractsync"
)

func main() {
	workdir := flag.String("workdir", ".", "Repository root to sync.")
	check := flag.Bool("check", false, "Verify generated contract artifacts are up to date.")
	flag.Parse()

	if err := contractsync.Sync(*workdir, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *check {
		fmt.Println("Contract schemas are in sync.")
		return
	}
	fmt.Println("Updated contract schemas.")
}
