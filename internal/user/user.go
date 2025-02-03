package user

import (
	"github.com/valyala/fasthttp"
)

type UserRole string

const (
	UserRoleOwner             UserRole = "owner"
	headerAuthorization                = "Authorization"
	headerAuthorizationBearer          = "Bearer"
)

type User struct {
	PublicKey string
	Role      UserRole
}

type UserRepository interface {
	CreateUser(user *User) error
}

func HandleUsersOwnersRegister(ctx *fasthttp.RequestCtx) {
	// TODO
	// authHeader := ctx.Request.Header.Peek(headerAuthorization)
	// if string(authHeader) != masterPassword {
	// 	ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
	// 	return
	// }

	// // Extract the public key from the request body
	// var requestBody struct {
	// 	PublicKey string `json:"public_key"`
	// }
	// if err := json.Unmarshal(ctx.Request.Body(), &requestBody); err != nil {
	// 	ctx.Error("Invalid request body", fasthttp.StatusBadRequest)
	// 	return
	// }

	// // Save the public key
	// mu.Lock()
	// ownerPublicKey = requestBody.PublicKey
	// mu.Unlock()

	// Respond with success
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("Public key saved successfully")
}

type OwnerConfig struct {
	MasterPasswordHash string `json:"master_password_hash"`
	RegistrationTTLSec int32  `json:"registration_ttl_sec"`
}
