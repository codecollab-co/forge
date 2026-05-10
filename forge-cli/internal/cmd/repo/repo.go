// Package repo holds the `forge repo` cobra command group.
package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codecollab-co/forge/forge-cli/internal/api"
	"github.com/codecollab-co/forge/forge-cli/internal/auth"
)

func New(stdout, stderr io.Writer, store *auth.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Create, list, view, and manage repositories",
	}
	cmd.AddCommand(listCmd(stdout, stderr, store))
	cmd.AddCommand(viewCmd(stdout, stderr, store))
	cmd.AddCommand(createCmd(stdout, stderr, store))
	cmd.AddCommand(deleteCmd(stdout, stderr, store))
	cmd.AddCommand(editCmd(stdout, stderr, store))
	cmd.AddCommand(cloneCmd(stdout, stderr, store))
	return cmd
}

// withClient resolves credentials and constructs an api.Client. Returns a
// helpful error if not logged in.
func withClient(store *auth.Store) (*api.Client, auth.Credentials, error) {
	c, err := store.Load()
	if errors.Is(err, auth.ErrNotLoggedIn) {
		return nil, c, errors.New("not logged in. Run `forge auth login`")
	}
	if err != nil {
		return nil, c, err
	}
	return api.New(c.APIURL, c.Token), c, nil
}

func parseSlug(slug string) (owner, name string, err error) {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected <owner>/<name>, got %q", slug)
	}
	return parts[0], parts[1], nil
}

func listCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var jsonOut bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List your repositories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli, _, err := withClient(store)
			if err != nil {
				return err
			}
			repos, err := cli.ListRepos(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(repos)
			}
			if len(repos) == 0 {
				fmt.Fprintln(stdout, "No repositories.")
				return nil
			}
			for _, r := range repos {
				vis := r.Visibility
				if vis == "" {
					vis = "public"
				}
				fmt.Fprintf(stdout, "%s/%s\t%s\t%s\n", r.Owner, r.Name, vis, r.Description)
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
		Use:   "view <owner>/<name>",
		Short: "Show a repository's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			cli, creds, err := withClient(store)
			if err != nil {
				return err
			}
			r, err := cli.GetRepo(cmd.Context(), owner, name)
			if err != nil {
				return err
			}
			if jsonOut {
				return json.NewEncoder(stdout).Encode(r)
			}
			fmt.Fprintf(stdout, "%s/%s (%s)\n", r.Owner, r.Name, r.Visibility)
			if r.Description != "" {
				fmt.Fprintf(stdout, "  %s\n", r.Description)
			}
			fmt.Fprintf(stdout, "  clone: %s%s\n", creds.APIURL, r.CloneURL)
			return nil
		},
	}
	c.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	return c
}

func createCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var (
		desc        string
		private     bool
		readme      bool
		importURL   string
	)
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new repository under your account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, creds, err := withClient(store)
			if err != nil {
				return err
			}
			vis := "public"
			if private {
				vis = "private"
			}
			r, err := cli.CreateRepo(cmd.Context(), api.CreateRepoInput{
				Name: args[0], Description: desc, Visibility: vis,
				InitReadme: readme, ImportURL: importURL,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Created %s/%s (%s)\n", r.Owner, r.Name, r.Visibility)
			fmt.Fprintf(stdout, "  clone: %s%s\n", creds.APIURL, r.CloneURL)
			return nil
		},
	}
	c.Flags().StringVarP(&desc, "description", "d", "", "Short description")
	c.Flags().BoolVar(&private, "private", false, "Create a private repository")
	c.Flags().BoolVar(&readme, "readme", false, "Initialize with a README.md")
	c.Flags().StringVar(&importURL, "import-url", "", "Import an existing Git URL (server-side mirror clone)")
	return c
}

func deleteCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var yes bool
	c := &cobra.Command{
		Use:   "delete <owner>/<name>",
		Short: "Delete a repository (irreversible)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			if !yes {
				return fmt.Errorf("refusing to delete %s/%s without --yes", owner, name)
			}
			cli, _, err := withClient(store)
			if err != nil {
				return err
			}
			if err := cli.DeleteRepo(cmd.Context(), owner, name); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Deleted %s/%s\n", owner, name)
			return nil
		},
	}
	c.Flags().BoolVar(&yes, "yes", false, "Confirm the destructive action")
	return c
}

func editCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var (
		desc       string
		visibility string
		rename     string
	)
	c := &cobra.Command{
		Use:   "edit <owner>/<name>",
		Short: "Edit a repository (description / visibility / rename)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			in := api.UpdateRepoInput{}
			if cmd.Flags().Changed("description") {
				in.Description = &desc
			}
			if cmd.Flags().Changed("visibility") {
				in.Visibility = &visibility
			}
			if cmd.Flags().Changed("rename") {
				in.Name = &rename
			}
			cli, _, err := withClient(store)
			if err != nil {
				return err
			}
			r, err := cli.UpdateRepo(cmd.Context(), owner, name, in)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Updated %s/%s (%s)\n", r.Owner, r.Name, r.Visibility)
			return nil
		},
	}
	c.Flags().StringVarP(&desc, "description", "d", "", "New description")
	c.Flags().StringVar(&visibility, "visibility", "", "public | private")
	c.Flags().StringVar(&rename, "rename", "", "New repository name")
	return c
}

// CloneExec is overridden in tests so we don't actually exec git.
var CloneExec = func(args ...string) error {
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func cloneCmd(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	c := &cobra.Command{
		Use:   "clone <owner>/<name> [target-dir]",
		Short: "git clone the repository (uses git, requires git credentials for private)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseSlug(args[0])
			if err != nil {
				return err
			}
			cli, creds, err := withClient(store)
			if err != nil {
				return err
			}
			r, err := cli.GetRepo(cmd.Context(), owner, name)
			if err != nil {
				return err
			}
			cloneURL := creds.APIURL + r.CloneURL
			// sanity-check it parses
			if _, err := url.Parse(cloneURL); err != nil {
				return fmt.Errorf("clone URL: %w", err)
			}
			gitArgs := []string{"clone", cloneURL}
			if len(args) == 2 {
				gitArgs = append(gitArgs, args[1])
			}
			fmt.Fprintf(stdout, "git %s\n", strings.Join(gitArgs, " "))
			return CloneExec(gitArgs...)
		},
	}
	return c
}

