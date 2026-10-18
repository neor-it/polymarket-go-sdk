package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/neor-it/polymarket-go-sdk/pkg/positions"
)

const exampleShareBaseUnits = 1000000

func main() {
	condition := os.Getenv("POLYMARKET_V2_CONDITION")
	if condition == "" {
		fmt.Println("Set POLYMARKET_V2_CONDITION to a native bytes31 or right-padded bytes32 V2 condition.")
		return
	}
	payload, err := positions.EncodeRedeem(condition, 0, big.NewInt(exampleShareBaseUnits))
	if err != nil {
		log.Fatal(err)
	}
	config := positions.ConfigPolygon137()
	fmt.Printf("Router: %s\nCalldata: 0x%s\n", config.Router.Hex(), hex.EncodeToString(payload))
	fmt.Println("Calldata only: no transaction was signed or broadcast.")
}
