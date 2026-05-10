package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/strait-dev/cli/internal/styles"
	"github.com/strait-dev/cli/internal/wizard"
)

// newMigrateCommand returns the `strait migrate` command, which converts
// existing job/workflow definitions from competing platforms (Inngest,
// Trigger.dev, Hatchet) into Strait `defineJob` / `defineWorkflow` TypeScript
// sources plus a `strait.deploy.json` manifest. Conversions are best-effort:
// fields that don't have a clean Strait analogue are emitted as `// TODO:
// review` comments.
func newMigrateCommand(state *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Convert jobs/workflows from another platform into Strait sources",
	}
	cmd.AddCommand(newMigrateInngestCommand(state))
	cmd.AddCommand(newMigrateTriggerCommand(state))
	cmd.AddCommand(newMigrateHatchetCommand(state))
	return cmd
}

// migratedJob is the platform-neutral shape every converter produces. The
// emitter renders one defineJob() block per entry and a manifest entry.
type migratedJob struct {
	Slug        string
	Name        string
	Description string
	EventTypes  []string // events this job listens to (best-effort)
	Notes       []string // free-form mapping notes surfaced as TODO comments
}

func newMigrateInngestCommand(state *appState) *cobra.Command {
	var input string
	var outDir string
	cmd := &cobra.Command{
		Use:   "inngest",
		Short: "Convert an Inngest functions export into Strait sources",
		Long: "Reads an Inngest functions JSON export (typically the output of " +
			"`inngest functions list --json` or the `inngest.json` config) and " +
			"emits `defineJob` TypeScript sources plus a `strait.deploy.json` " +
			"manifest. Conversion is best-effort; review the emitted TODO " +
			"comments before deploying.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			jobs, err := loadInngestJobs(input)
			if err != nil {
				return err
			}
			return emitMigratedJobs(cmd, state, "inngest", outDir, jobs)
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "path to the Inngest export JSON file")
	cmd.Flags().StringVar(&outDir, "out", "strait", "destination directory for emitted sources")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func newMigrateTriggerCommand(state *appState) *cobra.Command {
	var input string
	var outDir string
	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Convert a Trigger.dev jobs export into Strait sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			jobs, err := loadTriggerJobs(input)
			if err != nil {
				return err
			}
			return emitMigratedJobs(cmd, state, "trigger.dev", outDir, jobs)
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "path to the Trigger.dev jobs JSON export")
	cmd.Flags().StringVar(&outDir, "out", "strait", "destination directory for emitted sources")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func newMigrateHatchetCommand(state *appState) *cobra.Command {
	var input string
	var outDir string
	cmd := &cobra.Command{
		Use:   "hatchet",
		Short: "Convert a Hatchet workflow YAML into Strait sources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			jobs, err := loadHatchetJobs(input)
			if err != nil {
				return err
			}
			return emitMigratedJobs(cmd, state, "hatchet", outDir, jobs)
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "path to the Hatchet workflow YAML file")
	cmd.Flags().StringVar(&outDir, "out", "strait", "destination directory for emitted sources")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

// inngestExport mirrors the subset of Inngest's functions export we care
// about. Unknown fields are tolerated (encoding/json discards them).
type inngestExport struct {
	Functions []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Triggers []struct {
			Event string `json:"event"`
			Cron  string `json:"cron"`
		} `json:"triggers"`
	} `json:"functions"`
}

func loadInngestJobs(path string) ([]migratedJob, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path comes from --input CLI flag
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	var doc inngestExport
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse inngest JSON: %w", err)
	}
	jobs := make([]migratedJob, 0, len(doc.Functions))
	for _, fn := range doc.Functions {
		mj := migratedJob{
			Slug: normalizeSlug(firstNonEmpty(fn.ID, fn.Name)),
			Name: fn.Name,
		}
		for _, t := range fn.Triggers {
			if t.Event != "" {
				mj.EventTypes = append(mj.EventTypes, t.Event)
			}
			if t.Cron != "" {
				mj.Notes = append(mj.Notes, "cron trigger '"+t.Cron+"' — wire via `strait jobs schedule`")
			}
		}
		jobs = append(jobs, mj)
	}
	return jobs, nil
}

// triggerExport mirrors the Trigger.dev jobs export shape.
type triggerExport struct {
	Jobs []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Trigger struct {
			Type  string `json:"type"`
			Event struct {
				Name string `json:"name"`
			} `json:"event"`
			Cron string `json:"cron"`
		} `json:"trigger"`
	} `json:"jobs"`
}

func loadTriggerJobs(path string) ([]migratedJob, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path comes from --input CLI flag
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	var doc triggerExport
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse trigger.dev JSON: %w", err)
	}
	jobs := make([]migratedJob, 0, len(doc.Jobs))
	for _, j := range doc.Jobs {
		mj := migratedJob{
			Slug: normalizeSlug(firstNonEmpty(j.ID, j.Name)),
			Name: j.Name,
		}
		switch j.Trigger.Type {
		case "event":
			if j.Trigger.Event.Name != "" {
				mj.EventTypes = append(mj.EventTypes, j.Trigger.Event.Name)
			}
		case "scheduled", "cron":
			if j.Trigger.Cron != "" {
				mj.Notes = append(mj.Notes, "scheduled trigger '"+j.Trigger.Cron+"' — wire via `strait jobs schedule`")
			}
		default:
			if j.Trigger.Type != "" {
				mj.Notes = append(mj.Notes, "unsupported trigger type '"+j.Trigger.Type+"' — review manually")
			}
		}
		jobs = append(jobs, mj)
	}
	return jobs, nil
}

// hatchetExport mirrors the Hatchet workflow YAML shape.
type hatchetExport struct {
	Name     string `yaml:"name"`
	Triggers struct {
		Events []string `yaml:"events"`
		Cron   []string `yaml:"cron"`
	} `yaml:"triggers"`
	Jobs map[string]struct {
		Description string `yaml:"description"`
		Timeout     string `yaml:"timeout"`
	} `yaml:"jobs"`
}

func loadHatchetJobs(path string) ([]migratedJob, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path comes from --input CLI flag
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	var doc hatchetExport
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse hatchet YAML: %w", err)
	}
	if len(doc.Jobs) == 0 {
		return nil, fmt.Errorf("hatchet YAML has no `jobs:` entries")
	}
	// Sort job keys so output is deterministic.
	keys := make([]string, 0, len(doc.Jobs))
	for k := range doc.Jobs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	jobs := make([]migratedJob, 0, len(doc.Jobs))
	for _, key := range keys {
		spec := doc.Jobs[key]
		mj := migratedJob{
			Slug:        normalizeSlug(key),
			Name:        firstNonEmpty(doc.Name+"-"+key, key),
			Description: spec.Description,
			EventTypes:  append([]string(nil), doc.Triggers.Events...),
		}
		if spec.Timeout != "" {
			mj.Notes = append(mj.Notes, "hatchet timeout '"+spec.Timeout+"' — set via `strait jobs update --timeout`")
		}
		for _, c := range doc.Triggers.Cron {
			mj.Notes = append(mj.Notes, "cron trigger '"+c+"' — wire via `strait jobs schedule`")
		}
		jobs = append(jobs, mj)
	}
	return jobs, nil
}

// emitMigratedJobs writes one TypeScript source per migrated job under
// outDir/jobs/<slug>.ts plus a strait.deploy.json manifest at outDir/. The
// caller's platform name is included in the file header comments so users can
// trace where each block came from.
func emitMigratedJobs(cmd *cobra.Command, state *appState, platform, outDir string, jobs []migratedJob) error {
	if len(jobs) == 0 {
		return fmt.Errorf("no jobs found in input")
	}
	abs, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("resolve output dir: %w", err)
	}
	jobsDir := filepath.Join(abs, "jobs")
	if err := os.MkdirAll(jobsDir, 0o750); err != nil {
		return fmt.Errorf("create jobs dir: %w", err)
	}

	manifest := struct {
		Platform string          `json:"platform"`
		Jobs     []manifestEntry `json:"jobs"`
	}{Platform: platform}

	for _, job := range jobs {
		if err := wizard.ValidateSlug(job.Slug); err != nil {
			return fmt.Errorf("invalid slug %q (derived from %q): %w", job.Slug, job.Name, err)
		}
		path := filepath.Join(jobsDir, job.Slug+".ts")
		if err := os.WriteFile(path, []byte(renderJobTS(platform, job)), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		manifest.Jobs = append(manifest.Jobs, manifestEntry{
			Slug:       job.Slug,
			Source:     filepath.ToSlash(filepath.Join("jobs", job.Slug+".ts")),
			EventTypes: job.EventTypes,
		})
	}

	manifestPath := filepath.Join(abs, "strait.deploy.json")
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, append(manifestBytes, '\n'), 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	if isTTYRich(state) {
		fmt.Fprintln(cmd.ErrOrStderr(), styles.Success(fmt.Sprintf("Migrated %d job(s) from %s", len(jobs), platform)))
		fmt.Fprintln(cmd.ErrOrStderr(), styles.KeyValue("Output", abs))
		fmt.Fprintln(cmd.ErrOrStderr(), styles.KeyValue("Next", "review TODO comments, then `strait deploy push`"))
		return nil
	}
	return printData(state, map[string]any{
		"platform": platform,
		"out":      abs,
		"jobs":     len(jobs),
	})
}

type manifestEntry struct {
	Slug       string   `json:"slug"`
	Source     string   `json:"source"`
	EventTypes []string `json:"event_types,omitempty"`
}

// renderJobTS produces the TypeScript source for one migrated job. The
// generated module imports `defineJob` from the SDK, declares a placeholder
// `run` body, and surfaces every conversion note as a `// TODO: review`
// comment so the developer sees them before deploy.
func renderJobTS(platform string, job migratedJob) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// Migrated from %s\n", platform)
	if job.Name != "" && job.Name != job.Slug {
		fmt.Fprintf(&b, "// Source: %s\n", job.Name)
	}
	for _, n := range job.Notes {
		fmt.Fprintf(&b, "// TODO: review — %s\n", n)
	}
	for _, ev := range job.EventTypes {
		fmt.Fprintf(&b, "// Listens to event: %s\n", ev)
	}
	b.WriteString("\nimport { defineJob } from \"@strait/ts\";\n\n")
	fmt.Fprintf(&b, "export const %sJob = defineJob({\n", camelCase(job.Slug))
	fmt.Fprintf(&b, "  slug: %q,\n", job.Slug)
	if job.Description != "" {
		fmt.Fprintf(&b, "  description: %q,\n", job.Description)
	}
	b.WriteString("  run: async ({ payload }: { payload: Record<string, unknown> }) => {\n")
	b.WriteString("    // TODO: port the original handler logic.\n")
	b.WriteString("    return { migrated: true, payload };\n")
	b.WriteString("  },\n")
	b.WriteString("});\n")
	return b.String()
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case r == '-' || r == '_' || r == ' ' || r == '.' || r == '/':
			if !prevHyphen && b.Len() > 0 {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func camelCase(slug string) string {
	parts := strings.Split(slug, "-")
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(p)
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	out := b.String()
	if out == "" {
		return "migrated"
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
