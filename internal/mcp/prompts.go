package mcp

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mark3labs/mcp-go/mcp"
)

// Ensure skillDef and loadSkills are seen as used by linters that analyze
// per-package instead of whole-program.
var (
	_ = skillDef{}
	_ = (*Server).loadSkills
)

//go:embed skills/coding-agent.toml
var defaultSkillsFS embed.FS

type skillDef struct {
	Skill struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
	} `toml:"skill"`
}

func (s *Server) loadSkills(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".toml" {
			continue
		}
		fullPath := filepath.Join(dir, e.Name())
		var sk skillDef
		if _, err := toml.DecodeFile(fullPath, &sk); err != nil {
			fmt.Fprintf(os.Stderr, "mmrun: skipping skill %q: %v\n", e.Name(), err)
			continue
		}
		if sk.Skill.Name == "" || sk.Skill.Description == "" {
			fmt.Fprintf(os.Stderr, "mmrun: skipping skill %q: name and description are required\n", e.Name())
			continue
		}
		prompt := mcp.NewPrompt(sk.Skill.Name, mcp.WithPromptDescription(sk.Skill.Description))
		s.srv.AddPrompt(prompt, func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			return mcp.NewGetPromptResult(sk.Skill.Name, []mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(sk.Skill.Description)),
			}), nil
		})
	}
	return nil
}

func listSkills(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".toml" {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func initSkills(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	data, err := defaultSkillsFS.ReadFile("skills/coding-agent.toml")
	if err != nil {
		return fmt.Errorf("embedded skill not found: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "coding-agent.toml"), data, 0o644)
}

// ListSkills returns the names of skill TOML files in dir.
var ListSkills = listSkills

// InitSkills creates the skills directory and writes the embedded default skill.
var InitSkills = initSkills
