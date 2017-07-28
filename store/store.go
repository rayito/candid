// Copyright 2017 Canonical Ltd.

package store

import (
	"time"

	"golang.org/x/net/context"
	"gopkg.in/macaroon-bakery.v2-unstable/bakery"
)

// Field represents a field in an identity record.
type Field int

const (
	ProviderID Field = iota
	Username
	Name
	Email
	Groups
	PublicKeys
	LastLogin
	LastDischarge
	ProviderInfo
	ExtraInfo
	NumFields
)

// An Operation represents a type of update that can be applied to an
// identity record in a Store.UpdateIdentity call.
type Operation byte

const (
	// NoUpdate makes no changes to the field.
	NoUpdate Operation = iota

	// Set overrides the value of the field with the specified value.
	Set

	// Clear removes the field from the document.
	Clear

	// Push ensures that all the values in the field are added to any
	// that are already present.
	//
	// For the ProviderInfo and ExtraInfo fields the new values are
	// added to each specified key individually.
	Push

	// Pull ensures that all the values in the field are removed from
	// those present. It is legal to remove values that aren't
	// already stored.
	//
	// For the ProviderInfo and ExtraInfo fields the values are
	// removed from each specified key individually.
	Pull
)

// An Update is used in a Store.UpdateIdentity to specify how the
// identity record fields should be changed.
type Update [NumFields]Operation

// A Comparison represents a type of comparison that can be used in a
// filter in a Store.FindIdentities call.
type Comparison byte

const (
	NoComparison Comparison = iota
	Equal
	NotEqual
	GreaterThan
	LessThan
	GreaterThanOrEqual
	LessThanOrEqual
)

// A Filter is used in a Store.FindEntities call to specify how the
// identities should be filtered.
type Filter [NumFields]Comparison

// A Sort specifies the sort order of returned identities in a call to
// Store.FindIdenties.
type Sort struct {
	Field      Field
	Descending bool
}

// Store is the interface that represents the data storage mechanism for
// the identity manager.
type Store interface {
	// Identity reads the given identity from persistant storage and
	// completes all the fields. The given identity will be matched
	// using the first non-zero value of ID, ProviderID or Username.
	// If no match can found for the given identity then an error
	// with the cause ErrNotFound will be returned.
	Identity(ctx context.Context, identity *Identity) error

	// FindIdentities searches for all identities that match the
	// given ref when the given filter has been applied. The results
	// will be sorted in the order specified by sort. If limit is
	// greater than 0 then the results will contain at most that many
	// identities. If skip is greater than 0 then that many results
	// will be skipped before those that are returned.
	FindIdentities(ctx context.Context, ref *Identity, filter Filter, sort []Sort, skip, limit int) ([]Identity, error)

	// UpdateIdentity stores the data from the given identity in
	// persistant storage. The identity that is updated will be the
	// one matching the first non-zero value of ID, ProviderID or
	// Username. If the ID or username does not find a match then an
	// error with a cause of ErrNotFound will be returned. If there
	// is no match for an identity specified by ProviderID then a new
	// record will be created for the identity, in this case the
	// assigned ID will be written back into the given identity.
	//
	// The fields that are written to the database are dictateted by
	// the given UpdateOperations parameter. For each updatable field
	// this parameter will be consulted for the type of update to
	// perform. If the update would result in a duplicate username
	// being used then an error with the cause ErrDuplicateUsername
	// will be returned.
	UpdateIdentity(ctx context.Context, identity *Identity, update Update) error
}

// A ProviderIdentity is a provider-specific unique identity.
type ProviderIdentity string

// MakeProviderIdentity creates a ProviderIdentitiy from the given
// provider name and provider-specific identity.
func MakeProviderIdentity(provider, id string) ProviderIdentity {
	return ProviderIdentity(provider + ":" + id)
}

// Identity represents an identity in the store.
type Identity struct {
	// ID is the internal ID of the Identity, this is allocated by
	// the store when the identity is created.
	ID string

	// ProviderID contains the provider specific ID of the identity.
	ProviderID ProviderIdentity

	// Username contains the username of the identity.
	Username string

	// Name contains the display name of the identity.
	Name string

	// Email contains the email address of the identity.
	Email string

	// Groups contains the stored set of groups of which the identity
	// is a member.
	Groups []string

	// PublicKeys contains any public keys associated with the
	// identity.
	PublicKeys []bakery.PublicKey

	// LastLogin contains the time that the identity last logged in.
	LastLogin time.Time

	// LastDischarge contains the time that the identity last logged
	// in.
	LastDischarge time.Time

	// ProviderInfo contains provider specific information associated
	// with the identity. This field is reserved for the provider to
	// add any additional data the provider requires to manage the
	// identity.
	ProviderInfo map[string][]string

	// ExtraInfo contains extra information associated with the
	// identity. This field is used for any additional data that is
	// stored with the identity, but is not directly required by the
	// identity manager.
	ExtraInfo map[string][]string
}
