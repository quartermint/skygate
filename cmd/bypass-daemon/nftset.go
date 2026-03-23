package main

import "fmt"

const (
	nftTable  = "inet"
	nftFamily = "skygate"
	nftSet    = "bypass_v4"
)

// FormatNftCommand returns the arguments for an nft command to add an IP
// to the bypass set with the given timeout.
func FormatNftCommand(ip string, timeoutHours int) []string {
	element := fmt.Sprintf("{ %s timeout %dh }", ip, timeoutHours)
	return []string{"add", "element", nftTable, nftFamily, nftSet, element}
}
