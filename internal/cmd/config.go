package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/config"
	"github.com/isdmx/mmrun/internal/output"
)

func newConfigCmd(outputMode *string) *cobra.Command {
	cfg := &cobra.Command{
		Use: "config", Short: "View and edit configuration",
		Example: "  mmrun config set theme dark\n  mmrun config list",
	}

	cfg.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the configuration file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), config.Paths().ConfigFile)
			return nil
		},
	})

	cfg.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all settings with effective values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigList(*outputMode, cmd.OutOrStdout())
		},
	})

	cfg.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Print one setting's value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := runConfigGet(args[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), v)
			return nil
		},
	})

	cfg.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set and persist a setting",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1])
		},
	})

	var force bool
	gen := &cobra.Command{
		Use:   "generate",
		Short: "Write a commented default configuration file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := runConfigGenerate(force); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "wrote", config.Paths().ConfigFile)
			return nil
		},
	}
	gen.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")
	cfg.AddCommand(gen)

	return cfg
}

func runConfigGet(key string) (string, error) {
	c, err := config.Load()
	if err != nil {
		return "", err
	}
	return config.Get(c, key)
}

func runConfigSet(key, value string) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.Set(c, key, value); err != nil {
		return err
	}
	return config.Save(c)
}

func runConfigList(outputMode string, w io.Writer) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	res := output.Result{Title: "Configuration", Columns: []string{"key", "value", "description"}}
	for _, key := range config.Keys() {
		val, _ := config.Get(c, key)
		desc, _, _ := config.Describe(key)
		res.Rows = append(res.Rows, output.Row{"key": key, "value": val, "description": desc})
	}
	return output.New(outputMode, stdoutFile(w)).Render(w, res)
}

func runConfigGenerate(force bool) error {
	path := config.Paths().ConfigFile
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(config.Template()), 0o600)
}
