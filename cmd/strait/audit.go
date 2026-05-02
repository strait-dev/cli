package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/styles"

	"github.com/spf13/cobra"
)

func newAuditCommand(state *appState) *cobra.Command {
	var projectID string
	var actorID string
	var resourceType string
	var resourceID string
	var limit int
	var from string
	var to string
	var order string

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View audit log events",
		Long:  "Lists recent audit events for a project with optional filters.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			projectID, err = requireProjectID(state, projectID)
			if err != nil {
				return err
			}
			var fromTime *time.Time
			if from != "" {
				parsedFrom, parseErr := parseAuditTime(from)
				if parseErr != nil {
					return parseErr
				}
				fromTime = &parsedFrom
			}
			var toTime *time.Time
			if to != "" {
				parsedTo, parseErr := parseAuditTime(to)
				if parseErr != nil {
					return parseErr
				}
				toTime = &parsedTo
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			events, err := cli.ListAuditEvents(cmd.Context(), client.ListAuditEventsParams{
				ProjectID:    projectID,
				ActorID:      actorID,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Limit:        limit,
				From:         fromTime,
				To:           toTime,
				Order:        order,
			})
			if err != nil {
				return err
			}

			rows := make([]map[string]any, 0, len(events))
			for _, e := range events {
				action := e.Action
				if isTTYRich(state) {
					action = colorAuditAction(e.Action)
				}
				rows = append(rows, map[string]any{
					"id":            e.ID,
					"actor_id":      e.ActorID,
					"actor_type":    e.ActorType,
					"action":        action,
					"resource_type": e.ResourceType,
					"resource_id":   e.ResourceID,
					"details":       e.Details,
					"created_at":    e.CreatedAt,
				})
			}

			return printData(state, rows)
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&actorID, "actor-id", "", "filter by actor ID")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "filter by resource type")
	cmd.Flags().StringVar(&resourceID, "resource-id", "", "filter by resource ID")
	cmd.Flags().IntVar(&limit, "limit", 50, "max events to return")
	cmd.Flags().StringVar(&from, "from", "", "filter events created after this RFC3339 timestamp")
	cmd.Flags().StringVar(&to, "to", "", "filter events created before this RFC3339 timestamp")
	cmd.Flags().StringVar(&order, "order", "desc", "sort order (asc or desc)")

	cmd.AddCommand(newAuditVerifyCommand(state))

	return cmd
}

// auditVerifyResult is the CLI's stable JSON output shape for `audit verify`.
// It is intentionally decoupled from the server response so the CLI can expose
// a self-contained contract (status enum, first_break object, duration).
type auditVerifyResult struct {
	ProjectID     string            `json:"project_id"`
	Status        string            `json:"status"`
	EventsChecked int               `json:"events_checked"`
	FirstBreak    *auditVerifyBreak `json:"first_break"`
	DurationMS    int64             `json:"duration_ms"`
}

type auditVerifyBreak struct {
	EventID string `json:"event_id"`
	Reason  string `json:"reason"`
}

// errAuditChainBroken is returned when the server reports a broken chain.
// It propagates through cobra so the process exits non-zero.
var errAuditChainBroken = errors.New("audit chain verification failed")

func newAuditVerifyCommand(state *appState) *cobra.Command {
	var projectID string
	var since string
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify the audit event HMAC chain",
		Long: "Calls the server to verify the integrity of the project's audit event " +
			"HMAC chain. Exits 0 if the chain is intact, 1 if a break is detected.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedProject, err := requireProjectID(state, projectID)
			if err != nil {
				return err
			}

			format := strings.ToLower(strings.TrimSpace(outputFmt))
			if format == "" {
				format = "text"
			}
			if format != "text" && format != "json" {
				return fmt.Errorf("invalid --output %q: must be one of text, json", outputFmt)
			}

			var sinceTime *time.Time
			if strings.TrimSpace(since) != "" {
				parsed, parseErr := time.Parse(time.RFC3339Nano, since)
				if parseErr != nil {
					return fmt.Errorf("invalid --since value %q: must be RFC3339: %w", since, parseErr)
				}
				sinceTime = &parsed
			}

			cli, err := newAPIClient(state)
			if err != nil {
				return err
			}

			start := time.Now()
			resp, err := cli.VerifyAuditChain(cmd.Context(), client.VerifyAuditChainParams{
				ProjectID: resolvedProject,
				Since:     sinceTime,
			})
			duration := time.Since(start)
			if err != nil {
				return fmt.Errorf("verify audit chain: %w", err)
			}

			result := auditVerifyResult{
				ProjectID:     resp.ProjectID,
				EventsChecked: resp.EventsChecked,
				DurationMS:    duration.Milliseconds(),
			}
			if result.ProjectID == "" {
				result.ProjectID = resolvedProject
			}
			if resp.Valid {
				result.Status = "passed"
			} else {
				result.Status = "failed"
				reason := resp.Error
				if reason == "" {
					reason = "chain integrity broken"
				}
				result.FirstBreak = &auditVerifyBreak{
					EventID: resp.BrokenAtID,
					Reason:  reason,
				}
			}

			if err := renderAuditVerify(state, format, result); err != nil {
				return err
			}

			if !resp.Valid {
				return errAuditChainBroken
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectID, "project", "", "project ID (defaults to active context)")
	cmd.Flags().StringVar(&since, "since", "", "only verify events at or after this RFC3339 timestamp (optional)")
	cmd.Flags().StringVar(&outputFmt, "output", "text", "output format: text or json")

	return cmd
}

func renderAuditVerify(state *appState, format string, result auditVerifyResult) error {
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	rich := isTTYRich(state)
	var status string
	if rich {
		if result.Status == "passed" {
			status = styles.Green.Render("PASS")
		} else {
			status = styles.Red.Render("FAIL")
		}
	} else {
		if result.Status == "passed" {
			status = "PASS"
		} else {
			status = "FAIL"
		}
	}

	fmt.Fprintf(os.Stdout, "Audit chain verification for project %s: %s\n", result.ProjectID, status)
	fmt.Fprintf(os.Stdout, "  events checked: %d\n", result.EventsChecked)
	fmt.Fprintf(os.Stdout, "  duration:       %dms\n", result.DurationMS)
	if result.FirstBreak != nil {
		fmt.Fprintf(os.Stdout, "  first break:    event=%s reason=%s\n",
			dashIfEmpty(result.FirstBreak.EventID), result.FirstBreak.Reason)
	}
	return nil
}

func dashIfEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func colorAuditAction(action string) string {
	lower := strings.ToLower(action)
	switch {
	case strings.HasPrefix(lower, "create"):
		return styles.Green.Render(action)
	case strings.HasPrefix(lower, "delete"), strings.HasPrefix(lower, "revoke"):
		return styles.Red.Render(action)
	case strings.HasPrefix(lower, "update"), strings.HasPrefix(lower, "rotate"):
		return styles.Yellow.Render(action)
	default:
		return action
	}
}

func parseAuditTime(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}
