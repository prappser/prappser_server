package application

import "time"

type Application struct {
	ID              string           `json:"id"`
	OwnerPublicKey  string           `json:"ownerPublicKey"`
	UserPublicKey   string           `json:"userPublicKey"`
	Name            string           `json:"name"`
	CreatedAt       int64            `json:"createdAt"`
	UpdatedAt       int64            `json:"updatedAt"`
	ComponentGroups []ComponentGroup `json:"componentGroups,omitempty"`
	Members         []Member         `json:"members,omitempty"`
}

type ComponentGroup struct {
	ID            string      `json:"id"`
	ApplicationID string      `json:"applicationId"`
	Name          string      `json:"name"`
	Index         int         `json:"index"`
	Components    []Component `json:"components,omitempty"`
}

type Component struct {
	ID               string                 `json:"id"`
	ComponentGroupID string                 `json:"componentGroupId"`
	ApplicationID    string                 `json:"applicationId"`
	Name             string                 `json:"name"`
	Data             map[string]interface{} `json:"data,omitempty"`
	Index            int                    `json:"index"`
}

type ApplicationState struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt int64  `json:"updatedAt"`
}


type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
	MemberRoleViewer MemberRole = "viewer"
)

type Member struct {
	ID            string     `json:"id,omitempty"`
	ApplicationID string     `json:"applicationId"`
	Name          string     `json:"name"`
	Role          MemberRole `json:"role"`
	PublicKey     string     `json:"publicKey"`
	AvatarBytes   []byte     `json:"avatarBytes"`
}

func (a *Application) UpdateTimestamp() {
	a.UpdatedAt = time.Now().Unix()
}
