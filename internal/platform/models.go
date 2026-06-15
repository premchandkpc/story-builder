package platform

import (
	"time"

	"github.com/google/uuid"
)

type TenantID string

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
}

type Team struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
}

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleEditor  Role = "editor"
	RoleViewer  Role = "viewer"
	RoleOwner   Role = "owner"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Role           Role      `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

type APIKey struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	KeyHash        string    `json:"key_hash"`
	Name           string    `json:"name"`
	Scopes         []string  `json:"scopes"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}

type Permission string

const (
	PermReadStories      Permission = "stories:read"
	PermWriteStories     Permission = "stories:write"
	PermDeleteStories    Permission = "stories:delete"
	PermReadCharacters   Permission = "characters:read"
	PermWriteCharacters  Permission = "characters:write"
	PermGenerate         Permission = "generate"
	PermManageAPIKeys    Permission = "api_keys:manage"
	PermManageUsers      Permission = "users:manage"
)

type TenantContext struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id,omitempty"`
	Role           Role      `json:"role"`
}

type TenantStore interface {
	CreateOrganization(org *Organization) error
	GetOrganization(id uuid.UUID) (*Organization, error)
	CreateUser(user *User) error
	GetUser(id uuid.UUID) (*User, error)
	GetUserByEmail(email string) (*User, error)
	CreateAPIKey(key *APIKey) error
	GetAPIKey(keyHash string) (*APIKey, error)
	RevokeAPIKey(id uuid.UUID) error
	CheckPermission(orgID, userID uuid.UUID, permission Permission) bool
}
