package internal

import "github.com/prappser/prappser_server/internal/user"

type Config struct {
	Owners user.OwnerConfig `json:"owners"`
}
