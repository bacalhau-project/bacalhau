package publicapi

import (
	"crypto/rsa"
	"encoding/json"
	"net/http"

	jwk "github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/libp2p/go-libp2p/core/crypto"
)

// jwks godoc
//
//	@ID				apiServer/jwks
//	@Summary		Provides access to the node's public key via a JWKS Json file
//	@Description	Implements a JWKS endpoint (at a .well-known address) to deliver current public key for this node
//	@Tags			Misc
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	jwk.Set
//	@Failure		400				{object}	string
//	@Failure		500				{object}	string
//	@Router			/jwks [get]
//
//nolint:lll
func (apiServer *APIServer) jwks(res http.ResponseWriter, req *http.Request) {
	pubKey := apiServer.host.Peerstore().PubKey(apiServer.host.ID())

	key, err := crypto.PubKeyToStdKey(pubKey)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		http.Error(res, "key conversion error", http.StatusInternalServerError)
		return
	}

	jwkKey, err := jwk.FromRaw(rsaKey)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = jwkKey.Set(jwk.KeyIDKey, apiServer.host.ID().Pretty())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	jwkSet := jwk.NewSet()
	err = jwkSet.AddKey(jwkKey)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(jwkSet)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
