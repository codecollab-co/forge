// Package pr holds the `forge pr` cobra command group.
package pr

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
		Use:   "pr",
		Short: "Create, list, view, comment on, and merge Pull Requests",
	}
	cmd.AddCommand(listCmd(stdout, stderr, store))
	cmd.AddCommand(viewCmd(stdout, stderr, store))
	cmd.AddCommand(createCmd(stdout, stderr, store))
	cmd.AddCommand(commentCmd(stdout, stderr, store))
	cmd.AddCommand(mergeCmd(stdout, stderr, store))
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
		return 0, fmt.Errorf("invalid PR number: %q", s)
	}
	return n, nil
}

func listCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var state string
	var jsonOut bool
	c := &cobra.Command{
		Use:   "list <owner>/<name>",
		Short: "List Pull Requests on a repository",
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
			list, err := cli.ListPulls(cmd.Context(), owner, name, state)
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(list)
			}
			if len(list) == 0 {
				fmt.Fprintln(stdout, "No pull requests.")
				return nil
			}
			for _, pr := range list {
				fmt.Fprintf(stdout, "#%d\t%s\t%s\t%s → %s\t%s\n",
					pr.Number, pr.State, pr.Author, pr.HeadBranch, pr.BaseBranch, pr.Title)
			}
			return nil
		},
	}
	c.Flags().StringVarP(&state, "state", "s", "open", "open | merged | closed | all")
	c.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return c
}

func viewCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var jsonOut bool
	var showDiff bool
	c := &cobra.Command{
		Use:   "view <owner>/<name> <number>",
		Short: "Show a Pull Request's details",
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
			d, err := cli.GetPull(cmd.Context(), owner, name, n)
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(d)
			}
			pr := d.PullRequest
			fmt.Fprintf(stdout, "#%d %s [%s]\n", pr.Number, pr.Title, pr.State)
			fmt.Fprintf(stdout, "  @%s wants to merge %s → %s\n", pr.Author, pr.HeadBranch, pr.BaseBranch)
			if pr.MergeCommitOID != "" {
				fmt.Fprintf(stdout, "  merged as %s\n", pr.MergeCommitOID[:12])
			}
			if pr.Body != "" {
				fmt.Fprintf(stdout, "\n%s\n", pr.Body)
			}
			for _, c := range d.Comments {
				badge := ""
				if c.AuthorKind == "agent" {
					badge = " (agent)"
				}
				fmt.Fprintf(stdout, "\n--- @%s%s ---\n%s\n", c.Author, badge, c.Body)
			}
			if showDiff && d.Diff != "" {
				fmt.Fprintf(stdout, "\n--- diff ---\n%s\n", d.Diff)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	c.Flags().BoolVar(&showDiff, "diff", false, "Print the unified diff")
	return c
}

func createCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var title, body, head, base string
	c := &cobra.Command{
		Use:   "create <owner>/<name>",
		Short: "Open a new Pull Request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			if title == "" || head == "" || base == "" {
				return errors.New("--title, --head, and --base are required")
			}
			cli, err := withClient(store)
			if err != nil {
				return err
			}
			pr, err := cli.CreatePull(cmd.Context(), owner, name, api.CreatePullInput{
				Title: title, Body: body, HeadBranch: head, BaseBranch: base,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Opened #%d: %s (%s → %s)\n", pr.Number, pr.Title, pr.HeadBranch, pr.BaseBranch)
			return nil
		},
	}
	c.Flags().StringVarP(&title, "title", "t", "", "PR title (required)")
	c.Flags().StringVarP(&body, "body", "b", "", "PR body (Markdown)")
	c.Flags().StringVarP(&head, "head", "H", "", "Head branch (required)")
	c.Flags().StringVarP(&base, "base", "B", "", "Base branch (required)")
	return c
}

func commentCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var body string
	c := &cobra.Command{
		Use:   "comment <owner>/<name> <number>",
		Short: "Comment on a Pull Request",
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
			if _, err := cli.CommentOnPull(cmd.Context(), owner, name, n, body); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Commented on #%d.\n", n)
			return nil
		},
	}
	c.Flags().StringVarP(&body, "body", "b", "", "Comment body (required)")
	return c
}

func mergeCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var keepBranch bool
	c := &cobra.Command{
		Use:   "merge <owner>/<name> <number>",
		Short: "Merge a Pull Request",
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
			res, err := cli.MergePull(cmd.Context(), owner, name, n, !keepBranch)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Merged #%d as %s\n", n, res.MergeCommitOID[:12])
			return nil
		},
	}
	c.Flags().BoolVar(&keepBranch, "keep-branch", false, "Don't delete the head branch after merge (default: deleted)")
	return c
}
