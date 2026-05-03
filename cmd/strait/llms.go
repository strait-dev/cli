package main

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// llmsFlag describes a single CLI flag in the LLM manifest.
type llmsFlag struct {
	Name    string `json:"name"`
	Short   string `json:"short,omitempty"`
	Type    string `json:"type"`
	Default string `json:"default,omitempty"`
	Usage   string `json:"usage"`
}

// llmsCommand is a node in the LLM command-tree manifest.
type llmsCommand struct {
	Name        string        `json:"name"`
	Use         string        `json:"use"`
	Short       string        `json:"short"`
	Long        string        `json:"long,omitempty"`
	Example     string        `json:"example,omitempty"`
	Flags       []llmsFlag    `json:"flags,omitempty"`
	Subcommands []llmsCommand `json:"subcommands,omitempty"`
}

// llmsManifest is the top-level structure emitted by --llms.
type llmsManifest struct {
	CLI      string        `json:"cli"`
	Version  string        `json:"version"`
	Commands []llmsCommand `json:"commands"`
}

// printLLMSManifest writes the full command tree of root as compact JSON to w.
// Compact (no indentation) is intentional — this output is consumed by LLMs
// where token efficiency matters.
func printLLMSManifest(w io.Writer, root *cobra.Command) error {
	manifest := llmsManifest{
		CLI:      "strait",
		Version:  version,
		Commands: buildCommandTree(root.Commands()),
	}
	enc := json.NewEncoder(w)
	return enc.Encode(manifest)
}

// buildCommandTree recursively converts cobra commands to llmsCommand nodes,
// skipping hidden and completion-helper commands.
func buildCommandTree(cmds []*cobra.Command) []llmsCommand {
	out := make([]llmsCommand, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd.Hidden || cmd.Name() == "help" || cmd.Name() == "completion" {
			continue
		}
		node := llmsCommand{
			Name:  cmd.Name(),
			Use:   cmd.Use,
			Short: cmd.Short,
		}
		if cmd.Long != "" {
			// Trim to first paragraph to keep token count low.
			node.Long = firstParagraph(cmd.Long)
		}
		if cmd.Example != "" {
			node.Example = cmd.Example
		}

		// Local flags only — inherited flags (server, project, format, etc.)
		// are omitted per-command to avoid massive repetition.
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			flag := llmsFlag{
				Name:  f.Name,
				Short: f.Shorthand,
				Type:  f.Value.Type(),
				Usage: f.Usage,
			}
			if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
				flag.Default = f.DefValue
			}
			node.Flags = append(node.Flags, flag)
		})

		if subs := cmd.Commands(); len(subs) > 0 {
			node.Subcommands = buildCommandTree(subs)
		}

		out = append(out, node)
	}
	return out
}

// firstParagraph returns the text of s up to the first blank line.
func firstParagraph(s string) string {
	for i := range len(s) - 1 {
		if s[i] == '\n' && s[i+1] == '\n' {
			return s[:i]
		}
	}
	return s
}
