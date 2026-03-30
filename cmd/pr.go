package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	prBase  string
	prTitle string
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create a GitHub Pull Request using Claude to generate title and body",
	RunE:  runPR,
}

func init() {
	prCmd.Flags().StringVarP(&prBase, "base", "b", "", "base branch (default: repo default branch)")
	prCmd.Flags().StringVarP(&prTitle, "title", "t", "", "PR title (Claude generates if not provided)")
}

func runPR(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found. Install it from https://cli.github.com/")
	}

	// Check gh auth
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		return fmt.Errorf("gh CLI is not authenticated. Run 'gh auth login'")
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		return err
	}

	if currentBranch == "main" || currentBranch == "master" {
		return fmt.Errorf("cannot create PR from '%s' branch", currentBranch)
	}

	baseBranch := prBase
	if baseBranch == "" {
		baseBranch, err = getDefaultBranch()
		if err != nil {
			baseBranch = "main"
		}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " Analyzing changes with Claude..."
	s.Start()

	// Collect diffs and logs
	nameDiff, _ := runGitOutput("diff", "origin/"+baseBranch+"...HEAD", "--name-status")
	commitLog, _ := runGitOutput("log", "origin/"+baseBranch+"..HEAD", "--oneline")
	statDiff, _ := runGitOutput("diff", "origin/"+baseBranch+"...HEAD", "--stat")

	// Look for spec file
	specContent := findSpecForBranch(currentBranch)

	s.Stop()

	// Build prompt for Claude
	prompt := buildPRPrompt(currentBranch, baseBranch, nameDiff, commitLog, statDiff, specContent, prTitle)

	s2 := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s2.Suffix = " Creating PR with Claude..."
	s2.Start()
	defer s2.Stop()

	claudeCmd := exec.Command("claude", "--print", "--allowedTools", "Bash(gh:*)", prompt)
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr
	if err := claudeCmd.Run(); err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	s2.Stop()
	color.Green("✓ PR created successfully")
	return nil
}

func buildPRPrompt(branch, base, nameDiff, commitLog, statDiff, specContent, titleOverride string) string {
	titleInstruction := "Generate an appropriate PR title"
	if titleOverride != "" {
		titleInstruction = fmt.Sprintf("Use this PR title: %s", titleOverride)
	}

	specSection := ""
	if specContent != "" {
		specSection = fmt.Sprintf("\n\nSpec file content:\n%s", specContent)
	}

	return fmt.Sprintf(`Create a GitHub Pull Request using the gh CLI tool.

Branch: %s → %s

%s

Changed files:
%s

Commits:
%s

Stat:
%s
%s

Run: gh pr create --base %s --title "<title>" --body "<body>"

The body must include:
## Summary
- 2-3 bullet points describing what was done

## Test Plan
- [ ] checklist items for testing

🤖 Generated with Claude Code`,
		branch, base,
		titleInstruction,
		nameDiff,
		commitLog,
		statDiff,
		specSection,
		base,
	)
}

func findSpecForBranch(branch string) string {
	root, err := getRepoRoot()
	if err != nil {
		return ""
	}

	specsDir := filepath.Join(root, "specs")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return ""
	}

	// Convert branch name to likely spec filename
	// feature/my-feature -> my-feature.md
	parts := strings.SplitN(branch, "/", 2)
	branchBase := branch
	if len(parts) == 2 {
		branchBase = parts[1]
	}

	candidates := []string{
		filepath.Join(specsDir, branchBase+".md"),
		filepath.Join(specsDir, strings.ReplaceAll(branchBase, "-", "_")+".md"),
	}

	for _, path := range candidates {
		if content, err := os.ReadFile(path); err == nil {
			return string(content)
		}
	}

	// Try fuzzy match
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return ""
	}

	branchWords := strings.FieldsFunc(branchBase, func(r rune) bool {
		return r == '-' || r == '_'
	})

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		fileName := strings.TrimSuffix(entry.Name(), ".md")
		fileWords := strings.FieldsFunc(fileName, func(r rune) bool {
			return r == '-' || r == '_'
		})

		matches := 0
		for _, bw := range branchWords {
			for _, fw := range fileWords {
				if strings.EqualFold(bw, fw) {
					matches++
				}
			}
		}

		if matches >= len(branchWords)/2+1 || (len(branchWords) > 0 && matches == len(branchWords)) {
			content, err := os.ReadFile(filepath.Join(specsDir, entry.Name()))
			if err == nil {
				return string(content)
			}
		}
	}

	return ""
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]`)
var multiDashRe = regexp.MustCompile(`-+`)

func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = slugRe.ReplaceAllString(s, "")
	s = multiDashRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
