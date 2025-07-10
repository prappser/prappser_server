package owner

import (
	"database/sql"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

const (
	headerAuthorization       = "Authorization"
	headerAuthorizationBearer = "Bearer"
)

type Owner struct {
	PublicKey string
}

type OwnerRepository interface {
	CreateOwner(owner *Owner) error
}

type OwnerEndpoints struct {
	ownerRepository OwnerRepository
	config          Config
}

func NewSQLite3OwnerRepository(db *sql.DB) OwnerRepository {
	return &sqlite3OwnerRepository{db: db}
}

func NewEndpoints(ownerRepository OwnerRepository, config Config) *OwnerEndpoints {
	return &OwnerEndpoints{
		ownerRepository: ownerRepository,
		config:          config,
	}
}

func (oe OwnerEndpoints) OwnersRegister(ctx *fasthttp.RequestCtx) {
	authHeader := ctx.Request.Header.Peek(headerAuthorization)
	if authHeader == nil {
		log.Error().Msg("Missing authorization header")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}
	jwe, err := extractJWEFromAuthorizationHeader(string(authHeader))
	if err != nil {
		log.Error().Err(err).Msg("Invalid authorization header")
		ctx.Error("Invalid authorization header", fasthttp.StatusBadRequest)
		return
	}
	registerJWEClaims, err := decryptJWE(jwe, oe.config.MasterPasswordMD5Hash)
	if err != nil {
		log.Error().Err(err).Msg("Failed to decrypt JWE")
		ctx.Error("Failed to decrypt JWE", fasthttp.StatusBadRequest)
		return
	}
	registerJWSClaims, err := verifyJWS(registerJWEClaims.JWS, oe.config.RegistrationTokenTTLSec)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify JWS")
		ctx.Error("Failed to verify JWS", fasthttp.StatusBadRequest)
		return
	}
	newOwner := &Owner{
		PublicKey: registerJWSClaims.PublicKey,
	}
	err = oe.ownerRepository.CreateOwner(newOwner)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create owner")
		ctx.Error("Failed to create owner", fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("Owner registered successfully")
	return
}

type Config struct {
	MasterPasswordMD5Hash   string `mapstructure:"master_password_md5_hash"`
	RegistrationTokenTTLSec int32  `mapstructure:"registration_token_ttl_sec"`
}
