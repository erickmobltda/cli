package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec [name]",
	Short: "Create a structured spec file in the specs/ directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runSpec,
}

func runSpec(cmd *cobra.Command, args []string) error {
	if err := requireGitRepo(); err != nil {
		return err
	}

	name := args[0]
	slug := sanitizeSlug(name)

	root, err := getRepoRoot()
	if err != nil {
		return err
	}

	specsDir := filepath.Join(root, "specs")
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return fmt.Errorf("failed to create specs directory: %w", err)
	}

	specPath := filepath.Join(specsDir, slug+".md")

	if _, err := os.Stat(specPath); err == nil {
		return fmt.Errorf("spec file already exists: %s", specPath)
	}

	// Capitalize first letter of display name
	displayName := name
	if len(displayName) > 0 {
		displayName = strings.ToUpper(displayName[:1]) + displayName[1:]
	}

	content := fmt.Sprintf(`# %s

## Contexto
> Descreva o problema ou oportunidade que esta spec endereça.

## Objetivo
> O que queremos alcançar com esta implementação?

## Requisitos Funcionais
- [ ]
- [ ]
- [ ]

## Requisitos Não Funcionais
- [ ]
- [ ]

## Critérios de Aceite
- [ ]
- [ ]
- [ ]

## Fora de Escopo
> O que explicitamente NÃO será feito nesta iteração.

## Notas Técnicas
> Decisões de arquitetura, dependências relevantes, pontos de atenção.
`, displayName)

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write spec file: %w", err)
	}

	color.Green("✓ Spec created: %s", specPath)
	return nil
}
