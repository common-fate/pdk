package tokenstore

import (
	"github.com/99designs/keyring"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type Storage struct {
	keyring cfKeyring
}

var (
	ErrNotFound         = errors.New("auth token not found")
	AUTH_TOKEN_KEY_NAME = "commonfate_provider_registry"
)

// New creates a new token storage driver.
func New() Storage {
	return Storage{
		keyring: cfKeyring{},
	}
}

// keyname to store will always be "authtoken_commonfate_provider_registry"
func (s *Storage) key() string {
	return "authtoken_" + AUTH_TOKEN_KEY_NAME
}

// Clear the token
func (s *Storage) Clear() error {
	return s.keyring.Clear(s.key())
}

// Save the token
func (s *Storage) Save(token *oauth2.Token) error {
	return s.keyring.Store(s.key(), token)
}

// Token returns the OAuth2.0 token.
// It meets the TokenSource interface in the oauth2 package.
func (s *Storage) Token() (*oauth2.Token, error) {
	var t oauth2.Token
	err := s.keyring.Retrieve(s.key(), &t)
	if err == keyring.ErrKeyNotFound {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "keyring error")
	}

	return &t, nil
}
