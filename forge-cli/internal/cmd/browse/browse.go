// Package browse holds the `forge browse` command.
package browse

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codecollab-co/forge/forge-cli/internal/auth"
)

// Opener is the OS-specific browser opener. Overridden in tests.
var Opener = func(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

func New(stdout, _ io.Writer, store *auth.Store) *cobra.Command {
	var (
		issueN  int
		prN     int
		runID   string
		noBrowser bool
	)
	cmd := &cobra.Command{
		Use:   "browse [<owner>/<name>]",
		Short: "Open a Forge page in your browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			creds, err := store.Load()
			if errors.Is(err, auth.ErrNotLoggedIn) {
				return errors.New("not logged in. Run `forge auth login`")
			}
			if err != nil {
				return err
			}
			if creds.WebsiteURL == "" {
				return errors.New("website URL unknown — re-run `forge auth login`")
			}
			url := strings.TrimRight(creds.WebsiteURL, "/")
			if len(args) == 1 {
				owner, name, err := parseSlug(args[0])
				if err != nil {
					return err
				}
				url += "/" + owner + "/" + name
				switch {
				case issueN > 0:
					url += "/issues/" + strconv.Itoa(issueN)
				case prN > 0:
					url += "/pulls/" + strconv.Itoa(prN)
				}
			} else if runID != "" {
				url += "/runs/" + runID
			}
			if noBrowser {
				fmt.Fprintln(stdout, url)
				return nil
			}
			fmt.Fprintf(stdout, "Opening %s\n", url)
			return Opener(url)
		},
	}
	cmd.Flags().IntVar(&issueN, "issue", 0, "Open issue #N on the given repo")
	cmd.Flags().IntVar(&prN, "pr", 0, "Open pull request #N on the given repo")
	cmd.Flags().StringVar(&runID, "run", "", "Open the run trace page")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the URL instead of opening it")
	return cmd
}

func parseSlug(slug string) (string, string, error) {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected <owner>/<name>, got %q", slug)
	}
	return parts[0], parts[1], nil
}
