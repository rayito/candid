// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package local contains identity providers that validate against a builtin list of users.
// This provider is only intended for testing.
package local

import (
	"net/http"
	"strings"

	"golang.org/x/net/context"
	"gopkg.in/CanonicalLtd/candidclient.v1/params"
	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/httpbakery"

	"github.com/CanonicalLtd/candid/idp"
	"github.com/CanonicalLtd/candid/idp/idputil"
	"github.com/CanonicalLtd/candid/store"
)

func init() {
	idp.Register("local", func(unmarshal func(interface{}) error) (idp.IdentityProvider, error) {
		var p Params
		if err := unmarshal(&p); err != nil {
			return nil, errgo.Notef(err, "cannot unmarshal local parameters")
		}
		if p.Name == "" {
			p.Name = "local"
		}

		idp, err := NewIdentityProvider(p)
		if err != nil {
			return nil, errgo.Mask(err)
		}
		return idp, nil
	})
}

type Params struct {
	// Name is the name that will be given to the identity provider.
	Name string `yaml:"name"`

	// Domain is the domain with which all identities created by this
	// identity provider will be tagged (not including the @ separator).
	Domain string `yaml:"domain"`

	// Users is the set of users that are allowed to authenticate, with their
	// passwords and list of groups
	Users map[string]User `yaml:"users"`
}

type User struct {
	// Password is the password for the user.
	Password string `yaml:"password"`

	// Groups is the list of groups the user belongs to.
	Groups []string `yaml:"groups"`
}

// NewIdentityProvider creates a new local identity provider.
func NewIdentityProvider(p Params) (idp.IdentityProvider, error) {
	if len(p.Users) == 0 {
		return nil, errgo.Newf("no 'users' defined")
	}
	return &identityProvider{params: p}, nil

}

type identityProvider struct {
	params     Params
	initParams idp.InitParams
}

// Name implements idp.IdentityProvider.Name.
func (idp *identityProvider) Name() string {
	return idp.params.Name
}

// Domain implements idp.IdentityProvider.Domain.
func (idp *identityProvider) Domain() string {
	return idp.params.Domain
}

// Description implements idp.IdentityProvider.Description.
func (idp *identityProvider) Description() string {
	return "Local identity provider"
}

// Interactive implements idp.IdentityProvider.Interactive.
func (*identityProvider) Interactive() bool {
	return true
}

// Init implements idp.IdentityProvider.Init.
func (idp *identityProvider) Init(ctx context.Context, params idp.InitParams) error {
	idp.initParams = params
	return nil
}

// URL implements idp.IdentityProvider.URL.
func (idp *identityProvider) URL(dischargeID string) string {
	return idputil.URL(idp.initParams.URLPrefix, "/login", dischargeID)
}

// URL implements idp.IdentityProvider.SetInteraction.
func (idp *identityProvider) SetInteraction(ierr *httpbakery.Error, dischargeID string) {
}

//  GetGroups implements idp.IdentityProvider.GetGroups.
func (idp *identityProvider) GetGroups(ctx context.Context, identity *store.Identity) ([]string, error) {
	_, fulluser := identity.ProviderID.Split()
	username := strings.SplitN(fulluser, "@", 2)[0]
	if user, ok := idp.params.Users[username]; ok {
		return user.Groups, nil
	}
	return []string{}, nil
}

// Handle implements idp.IdentityProvider.Handle.
func (idp *identityProvider) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	switch strings.TrimPrefix(req.URL.Path, idp.initParams.URLPrefix) {
	case "/login":
		if err := idp.handleLogin(ctx, w, req); err != nil {
			idp.initParams.VisitCompleter.Failure(ctx, w, req, idputil.DischargeID(req), err)
		}
	}
}

func (idp *identityProvider) handleLogin(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	switch req.Method {
	default:
		return errgo.WithCausef(nil, params.ErrBadRequest, "unsupported method %q", req.Method)
	case "GET":
		return errgo.Mask(idp.initParams.Template.ExecuteTemplate(w, "login-form", nil))
	case "POST":
		id, err := idp.loginUser(ctx, req.Form.Get("username"), req.Form.Get("password"))
		if err != nil {
			return err
		}
		idp.initParams.VisitCompleter.Success(ctx, w, req, idputil.DischargeID(req), id)
		return nil
	}
}

func (idp *identityProvider) loginUser(ctx context.Context, user, password string) (*store.Identity, error) {
	if userData, ok := idp.params.Users[user]; ok {
		if userData.Password == password {
			username := idputil.NameWithDomain(user, idp.params.Domain)
			id := &store.Identity{
				ProviderID: store.MakeProviderIdentity(idp.params.Name, username),
				Username:   username,
				Name:       username,
				Email:      username,
			}
			err := idp.initParams.Store.UpdateIdentity(ctx, id, store.Update{
				store.Username: store.Set,
				store.Name:     store.Set,
				store.Email:    store.Set,
			})
			if err != nil {
				return nil, errgo.Mask(err)
			}
			return id, nil
		}
	}
	return nil, errgo.Newf("authentication failed for user %q", user)
}
