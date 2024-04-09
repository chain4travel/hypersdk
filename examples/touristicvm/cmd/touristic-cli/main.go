// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "touristic-cli" implements touristicvm client operation interface.
package main

import (
	"os"

	"github.com/ava-labs/hypersdk/utils"
	"github.com/chain4travel/hypersdk/examples/touristicvm/cmd/touristic-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		utils.Outf("{{red}}touristic-cli exited with error:{{/}} %+v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
