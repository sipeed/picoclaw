// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/update"
)

func updateCmd() {
	// Handle flags
	checkOnly := false
	for _, arg := range os.Args[2:] {
		switch arg {
		case "--help", "-h":
			updateHelp()
			return
		case "--check", "-c":
			checkOnly = true
		}
	}

	fmt.Printf("%s Checking for updates...\n", logo)

	release, err := update.CheckLatest()
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	latestVersion := release.TagName

	if !update.IsNewer(version, latestVersion) {
		fmt.Printf("Already up to date! (current: %s, latest: %s)\n", version, latestVersion)
		return
	}

	fmt.Printf("New version available: %s (current: %s)\n", latestVersion, version)
	fmt.Printf("Release: %s\n", release.HTMLURL)

	if checkOnly {
		return
	}

	assetURL, err := update.FindAssetURL(release)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("You can download manually from: %s\n", release.HTMLURL)
		os.Exit(1)
	}

	fmt.Printf("\nUpdate to %s? (y/n): ", latestVersion)
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" {
		fmt.Println("Update cancelled.")
		return
	}

	if err := update.DownloadAndReplace(assetURL, os.Stdout); err != nil {
		fmt.Printf("Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n%s Updated to %s!\n", logo, latestVersion)
}

func updateHelp() {
	fmt.Println("Check for updates and self-update picoclaw")
	fmt.Println()
	fmt.Println("Usage: picoclaw update [flags]")
	fmt.Println()
	fmt.Println("Checks GitHub for the latest release and offers to update in-place.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -c, --check    Check only, don't prompt to update")
	fmt.Println("  -h, --help     Show this help")
}
