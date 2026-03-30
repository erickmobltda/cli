package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Rebase current branch on top of origin/main",
	RunE:  runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	if currentBranch == "main" || currentBranch == "master" {
		return fmt.Errorf("already on %s, nothing to sync", currentBranch)
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Fetching origin/main..."
	s.Start()

	fetchErr := exec.Command("git", "fetch", "origin", "main").Run()
	if fetchErr != nil {
		// Try master
		fetchErr = exec.Command("git", "fetch", "origin", "master").Run()
	}
	s.Stop()

	if fetchErr != nil {
		color.Yellow("⚠ Could not fetch from origin: %v", fetchErr)
	} else {
		color.Green("✓ Fetched origin/main")
	}

	// Get commit count before rebase
	countBefore, _ := runGitOutput("rev-list", "--count", "HEAD")

	color.Cyan("→ Rebasing %s onto origin/main...", currentBranch)

	rebaseCmd := exec.Command("git", "rebase", "origin/main")
	rebaseOutput, err := rebaseCmd.CombinedOutput()

	if err != nil {
		// Abort rebase on conflict
		exec.Command("git", "rebase", "--abort").Run()
		color.Red("✗ Rebase failed with conflicts:")
		fmt.Println(string(rebaseOutput))
		color.Yellow("\nRebase aborted. Please resolve conflicts manually:")
		color.Yellow("  1. git fetch origin main")
		color.Yellow("  2. git rebase origin/main")
		color.Yellow("  3. Resolve conflicts, then: git rebase --continue")
		return fmt.Errorf("rebase failed")
	}

	// Get commit count and new HEAD
	countAfter, _ := runGitOutput("rev-list", "--count", "HEAD")
	newHead, _ := runGitOutput("rev-parse", "--short", "HEAD")

	_ = countBefore
	_ = strings.TrimSpace(countAfter)

	color.Green("✓ Rebase successful")
	color.Green("  New HEAD: %s", newHead)

	return nil
}
