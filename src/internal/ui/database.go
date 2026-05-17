//go:build cli

package ui

import (
	"encoding/json"
	"fmt"

	"marabu/internal/core"
	"marabu/internal/types"
)

func inspectObject(hashStr string, manager *core.Manager) {
	hash := types.HashID(hashStr)

	obj, err := manager.GetObject(hash)
	if err != nil {
		fmt.Printf("%sError: Could not find object %s in database.%s\n", red, hashStr, reset)
		return
	}

	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Printf("%sError formatting object: %v%s\n", red, err, reset)
		return
	}

	fmt.Printf("\n%s--- Object %s ---%s\n", bold+cyan, hashStr, reset)
	fmt.Println(green + string(prettyJSON) + reset)
	fmt.Printf("%s----------------------------------------------------------------%s\n\n", bold+cyan, reset)
}

func listObjects(manager *core.Manager) {
	ids, err := manager.GetAllObjectIDs()
	if err != nil {
		fmt.Printf("%sError scanning database: %v%s\n", red, err, reset)
		return
	}

	if len(ids) == 0 {
		fmt.Printf("\n%sNo objects found in the database.%s\n\n", yellow, reset)
		return
	}

	var blocks []types.HashID
	var txs []types.HashID
	var unknown []types.HashID

	fmt.Printf("%sScanning database...%s\n", yellow, reset)

	for _, id := range ids {
		obj, err := manager.GetObject(id)
		if err != nil {
			unknown = append(unknown, id)
			continue
		}

		switch obj.ObjectType() {
		case types.OBJ_BLOCK:
			blocks = append(blocks, id)
		case types.OBJ_TRANSACTION:
			txs = append(txs, id)
		default:
			unknown = append(unknown, id)
		}
	}

	fmt.Printf("\n%s=== Stored Objects ===%s\n", bold+cyan, reset)
	fmt.Printf("%sTotal: %d%s\n\n", bold, len(ids), reset)

	fmt.Printf("%sBlocks (%d):%s\n", bold+magenta, len(blocks), reset)
	for _, b := range blocks {
		fmt.Printf("  | %s%s%s\n", magenta, b, reset)
	}

	fmt.Printf("\n%sTransactions (%d):%s\n", bold+green, len(txs), reset)
	for _, t := range txs {
		fmt.Printf("  | %s%s%s\n", green, t, reset)
	}

	if len(unknown) > 0 {
		fmt.Printf("\n%sUnknown/Corrupted (%d):%s\n", bold+red, len(unknown), reset)
		for _, u := range unknown {
			fmt.Printf("    %s%s%s\n", red, u, reset)
		}
	}
	fmt.Printf("%s======================%s\n\n", bold+cyan, reset)
}
