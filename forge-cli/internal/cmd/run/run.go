// Package run holds the `forge run` cobra command group.
package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/codecollab-co/forge/forge-cli/internal/api"
	"github.com/codecollab-co/forge/forge-cli/internal/auth"
)

func New(stdout, stderr io.Writer, store *auth.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "List, view, watch, and cancel agent runs",
	}
	cmd.AddCommand(listCmd(stdout, stderr, store))
	cmd.AddCommand(viewCmd(stdout, stderr, store))
	cmd.AddCommand(cancelCmd(stdout, stderr, store))
	cmd.AddCommand(watchCmd(stdout, stderr, store))
	return cmd
}

func withClient(store *auth.Store) (*api.Client, error) {
	c, err := store.Load()
	if errors.Is(err, auth.ErrNotLoggedIn) {
		return nil, errors.New("not logged in. Run `forge auth login`")
	}
	if err != nil {
		return nil, err
	}
	return api.New(c.APIURL, c.Token), nil
}

func listCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List your recent runs across all repositories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			runs, err := cli.ListMyRuns(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(runs)
			}
			if len(runs) == 0 {
				fmt.Fprintln(stdout, "No runs.")
				return nil
			}
			for _, r := range runs {
				prCol := "-"
				if r.PRNumber != nil {
					prCol = fmt.Sprintf("PR #%d", *r.PRNumber)
				}
				fmt.Fprintf(stdout, "%s\t%s\t%s/%s#%d\t%s\t%s\n",
					r.ID[:8], r.State, r.RepoOwner, r.RepoName, r.IssueNumber, prCol, r.IssueTitle)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return c
}

func viewCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "view <run-id>",
		Short: "Show a run's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			r, err := cli.GetRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(r)
			}
			fmt.Fprintf(stdout, "Run %s\n  state: %s\n", r.ID, r.State)
			if r.SandboxID != "" {
				fmt.Fprintf(stdout, "  sandbox: %s\n", r.SandboxID)
			}
			if r.PRNumber != "" {
				fmt.Fprintf(stdout, "  pr: #%s\n", r.PRNumber)
			}
			if r.ErrorMessage != "" {
				fmt.Fprintf(stdout, "  error (%s): %s\n", r.ErrorCategory, r.ErrorMessage)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return c
}

func cancelCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <run-id>",
		Short: "Request cancellation of a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			if err := cli.CancelRun(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Cancellation requested for run %s.\n", args[0])
			return nil
		},
	}
}

func watchCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "watch <run-id>",
		Short: "Tail a run's events live (SSE)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			r, err := cli.GetRun(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if r.StreamURL == "" {
				return errors.New("run has no stream_url (server may be misconfigured)")
			}
			fmt.Fprintf(stdout, "Watching %s (state=%s)…\n", r.ID, r.State)
			ch, err := cli.StreamRun(cmd.Context(), r.StreamURL)
			if err != nil {
				return err
			}
			for ev := range ch {
				fmt.Fprintf(stdout, "[%s] %s %s\n", ev.ID, ev.Type, ev.Payload)
				if ev.Type == "run.terminal" {
					return nil
				}
			}
			return nil
		},
	}
}
