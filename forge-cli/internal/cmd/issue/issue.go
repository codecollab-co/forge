// Package issue holds the `forge issue` cobra command group.
package issue

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codecollab-co/forge/forge-cli/internal/api"
	"github.com/codecollab-co/forge/forge-cli/internal/auth"
)

func New(stdout, stderr io.Writer, store *auth.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Create, list, view, and manage Issues",
	}
	cmd.AddCommand(listCmd(stdout, stderr, store))
	cmd.AddCommand(viewCmd(stdout, stderr, store))
	cmd.AddCommand(createCmd(stdout, stderr, store))
	cmd.AddCommand(commentCmd(stdout, stderr, store))
	cmd.AddCommand(stateCmd(stdout, stderr, store, "close"))
	cmd.AddCommand(stateCmd(stdout, stderr, store, "reopen"))
	cmd.AddCommand(assignAgentCmd(stdout, stderr, store))
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

func parseSlug(slug string) (string, string, error) {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected <owner>/<name>, got %q", slug)
	}
	return parts[0], parts[1], nil
}

func parseNumber(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid issue number: %q", s)
	}
	return n, nil
}

func listCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var state string
	var jsonOut bool
	c := &cobra.Command{
		Use:   "list <owner>/<name>",
		Short: "List Issues on a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			list, err := cli.ListIssues(cmd.Context(), owner, name, state)
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(list)
			}
			if len(list) == 0 {
				fmt.Fprintln(stdout, "No issues.")
				return nil
			}
			for _, i := range list {
				assignee := ""
				if i.Assignee != nil {
					assignee = " @" + i.Assignee.Handle
				}
				fmt.Fprintf(stdout, "#%d\t%s\t%s\t%s%s\n", i.Number, i.State, i.Author, i.Title, assignee)
			}
			return nil
		},
	}
	c.Flags().StringVarP(&state, "state", "s", "open", "open | closed | all")
	c.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return c
}

func viewCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "view <owner>/<name> <number>",
		Short: "Show an Issue's details and comments",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			n, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			d, err := cli.GetIssue(cmd.Context(), owner, name, n)
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(d)
			}
			fmt.Fprintf(stdout, "#%d %s [%s]\n", d.Issue.Number, d.Issue.Title, d.Issue.State)
			fmt.Fprintf(stdout, "  opened by @%s\n", d.Issue.Author)
			if d.Issue.Body != "" {
				fmt.Fprintf(stdout, "\n%s\n", d.Issue.Body)
			}
			for _, c := range d.Comments {
				fmt.Fprintf(stdout, "\n--- @%s ---\n%s\n", c.Author, c.Body)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return c
}

func createCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var title, body string
	c := &cobra.Command{
		Use:   "create <owner>/<name>",
		Short: "Open a new Issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			if strings.TrimSpace(title) == "" {
				return errors.New("--title is required")
			}
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			i, err := cli.CreateIssue(cmd.Context(), owner, name, api.CreateIssueInput{Title: title, Body: body})
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Opened #%d: %s\n", i.Number, i.Title)
			return nil
		},
	}
	c.Flags().StringVarP(&title, "title", "t", "", "Issue title (required)")
	c.Flags().StringVarP(&body, "body", "b", "", "Issue body (Markdown)")
	return c
}

func commentCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var body string
	c := &cobra.Command{
		Use:   "comment <owner>/<name> <number>",
		Short: "Comment on an Issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			n, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			if strings.TrimSpace(body) == "" {
				return errors.New("--body is required")
			}
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			if _, err := cli.CommentOnIssue(cmd.Context(), owner, name, n, body); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Commented on #%d.\n", n)
			return nil
		},
	}
	c.Flags().StringVarP(&body, "body", "b", "", "Comment body (required)")
	return c
}

func stateCmd(stdout, _ io.Writer, store *auth.Store, op string) *cobra.Command {
	return &cobra.Command{
		Use:   op + " <owner>/<name> <number>",
		Short: strings.Title(op) + " an Issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			n, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			var i api.Issue
			if op == "close" {
				i, err = cli.CloseIssue(cmd.Context(), owner, name, n)
			} else {
				i, err = cli.ReopenIssue(cmd.Context(), owner, name, n)
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "#%d is now %s.\n", i.Number, i.State)
			return nil
		},
	}
}

func assignAgentCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "assign-agent <owner>/<name> <number>",
		Short: "Assign an Issue to the Forge agent (creates a Run)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			n, err := parseNumber(args[1])
			if err != nil {
				return err
			}
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			r, err := cli.AssignAgent(cmd.Context(), owner, name, n)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Run %s queued. Tail with: forge run watch %s\n", r.ID, r.ID)
			return nil
		},
	}
}
