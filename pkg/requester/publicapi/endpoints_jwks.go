package publicapi

import (
	"crypto/rsa"
	"encoding/json"
	"net/http"

	jwk "github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func (s *RequesterAPIServer) jwks(res http.ResponseWriter, req *http.Request) {
	pubKey := s.host.Peerstore().PubKey(s.host.ID())
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

	err = jwkKey.Set(jwk.KeyIDKey, s.host.ID().Pretty())
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
