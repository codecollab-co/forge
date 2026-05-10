// Package auth persists CLI credentials to disk.
//
// MVP: a JSON file at <configDir>/credentials.json with mode 0600. OS keychain
// integration is a v0.2 follow-up — interface stays the same.
package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var ErrNotLoggedIn = errors.New("not logged in")

type Credentials struct {
	APIURL     string `json:"api_url"`
	WebsiteURL string `json:"website_url,omitempty"`
	Token      string `json:"token"`
	Handle     string `json:"handle"`
}

type Store struct {
	dir string
}

func NewStore(configDir string) *Store {
	return &Store{dir: configDir}
}

func (s *Store) path() string {
	return filepath.Join(s.dir, "credentials.json")
}

func (s *Store) Load() (Credentials, error) {
	body, err := os.ReadFile(s.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, ErrNotLoggedIn
		}
		return Credentials{}, err
	}
	var c Credentials
	if err := json.Unmarshal(body, &c); err != nil {
		return Credentials{}, err
	}
	if c.Token == "" {
		return Credentials{}, ErrNotLoggedIn
	}
	return c, nil
}

func (s *Store) Save(c Credentials) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), body, 0o600)
}

func (s *Store) Clear() error {
	if err := os.Remove(s.path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
