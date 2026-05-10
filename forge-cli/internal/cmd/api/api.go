// Package api holds the `forge api` command — a raw HTTP passthrough.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	apiclient "github.com/codecollab-co/forge/forge-cli/internal/api"
	"github.com/codecollab-co/forge/forge-cli/internal/auth"
)

func New(stdout, stderr io.Writer, store *auth.Store) *cobra.Command {
	var (
		method string
		fields []string
	)
	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Make an authenticated request to the Forge API and print the body",
		Long: `Make an authenticated HTTP request to forge-platform. Default method
is GET; use -X/--method to change it. -F/--field "key=value" builds a JSON
request body. Use "-" as the path to leave the path blank, or pass any
"/path" relative to the API base URL. Pipe a body in via stdin to send it
verbatim.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := store.Load()
			if errors.Is(err, auth.ErrNotLoggedIn) {
				return errors.New("not logged in. Run `forge auth login`")
			}
			if err != nil {
				return err
			}
			cli := apiclient.New(creds.APIURL, creds.Token)

			var body io.Reader
			if len(fields) > 0 {
				obj := map[string]any{}
				for _, f := range fields {
					k, v, ok := strings.Cut(f, "=")
					if !ok {
						return fmt.Errorf("--field must be key=value, got %q", f)
					}
					obj[k] = parseFieldValue(v)
				}
				raw, _ := json.Marshal(obj)
				body = strings.NewReader(string(raw))
			} else if stdinHasData() {
				body = os.Stdin
			}

			status, err := cli.RawRequest(cmd.Context(), method, args[0], body, stdout)
			if err != nil {
				return err
			}
			if status >= 400 {
				return fmt.Errorf("HTTP %d", status)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	cmd.Flags().StringArrayVarP(&fields, "field", "F", nil, "key=value pairs collected into a JSON body")
	return cmd
}

// parseFieldValue tries to recognise booleans / numbers, otherwise leaves the
// value as a string. Mirrors gh's behaviour roughly.
func parseFieldValue(v string) any {
	switch v {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	return v
}

func stdinHasData() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Either piped-in data or a regular file (test redirection).
	return (st.Mode() & os.ModeCharDevice) == 0
}
