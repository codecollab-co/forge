// Package auth holds the `forge auth` cobra command group.
package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/codecollab-co/forge/forge-cli/internal/api"
	authstore "github.com/codecollab-co/forge/forge-cli/internal/auth"
)

// PollClock controls the device-token polling interval. Override in tests.
var PollClock = func(d time.Duration) <-chan time.Time { return time.After(d) }

func New(stdout, stderr io.Writer, store *authstore.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Sign in / out / inspect credentials",
	}
	cmd.AddCommand(loginCmd(stdout, stderr, store))
	cmd.AddCommand(statusCmd(stdout, stderr, store))
	cmd.AddCommand(logoutCmd(stdout, stderr, store))
	return cmd
}

func loginCmd(stdout, _ io.Writer, store *authstore.Store) *cobra.Command {
	var apiURL string
	c := &cobra.Command{
		Use:   "login",
		Short: "Authorize this device with Forge (RFC 8628 device-code flow)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client := api.New(apiURL, "")
			dc, err := client.RequestDeviceCode(cmd.Context())
			if err != nil {
				return fmt.Errorf("requesting device code: %w", err)
			}
			fmt.Fprintf(stdout, "Open %s and enter the code:\n\n    %s\n\n", dc.VerificationURI, dc.UserCode)
			interval := time.Duration(dc.Interval) * time.Second
			if interval == 0 {
				interval = 5 * time.Second
			}
			deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
			for time.Now().Before(deadline) {
				select {
				case <-cmd.Context().Done():
					return cmd.Context().Err()
				case <-PollClock(interval):
				}
				tok, err := client.PollDeviceToken(cmd.Context(), dc.DeviceCode)
				if errors.Is(err, api.ErrAuthorizationPending) {
					continue
				}
				if err != nil {
					return fmt.Errorf("polling for token: %w", err)
				}
				// Best-effort fetch of website URL for `forge browse`. Non-fatal.
				cfg, _ := api.New(apiURL, "").GetConfig(cmd.Context())
				if err := store.Save(authstore.Credentials{
					APIURL: apiURL, WebsiteURL: cfg.WebsiteURL,
					Token: tok.AccessToken, Handle: tok.Handle,
				}); err != nil {
					return fmt.Errorf("saving credentials: %w", err)
				}
				fmt.Fprintf(stdout, "Signed in as @%s.\n", tok.Handle)
				return nil
			}
			return errors.New("device code expired before approval")
		},
	}
	c.Flags().StringVar(&apiURL, "api-url", "http://localhost:8080", "forge-platform base URL")
	return c
}

func statusCmd(stdout, _ io.Writer, store *authstore.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := store.Load()
			if errors.Is(err, authstore.ErrNotLoggedIn) {
				fmt.Fprintln(stdout, "Not logged in. Run `forge auth login`.")
				return nil
			}
			if err != nil {
				return err
			}
			// Verify the token still works.
			cli := api.New(c.APIURL, c.Token)
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()
			me, err := cli.Me(ctx)
			if err != nil {
				fmt.Fprintf(stdout, "Stored credentials for @%s on %s — but the server rejected them: %v\n", c.Handle, c.APIURL, err)
				return nil
			}
			fmt.Fprintf(stdout, "Signed in as @%s on %s.\n", me.Handle, c.APIURL)
			return nil
		},
	}
}

func logoutCmd(stdout, _ io.Writer, store *authstore.Store) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Forget the current credentials",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := store.Clear(); err != nil {
				return err
			}
			fmt.Fprintln(stdout, "Logged out.")
			return nil
		},
	}
}
