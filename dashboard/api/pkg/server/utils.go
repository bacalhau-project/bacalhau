package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/model"
	"github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/golang-jwt/jwt"
	"github.com/rs/zerolog/log"
)

type httpErrorFunc func(http.ResponseWriter, *http.Request) error
type handlerFunc[Accepts, Returns any] func(context.Context, Accepts) (Returns, error)
type httpTypeFunc[Returns any] handlerFunc[*http.Request, Returns]

type userContextKey struct{}

func GetRequestBody[T any](r *http.Request) (requestBody *T, err error) {
	requestBody = new(T)
	if r.Body == nil {
		err = fmt.Errorf("no json to decode from empty request body: %+v", r)
	} else {
		err = json.NewDecoder(r.Body).Decode(requestBody)
	}
	return requestBody, err
}

func handleError(handler httpErrorFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		err := handler(res, req)
		if err != nil {
			log.Ctx(req.Context()).Error().Stringer("route", req.URL).Err(err).Send()
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	}
}

func returnsJSON[T any](handler httpTypeFunc[T]) httpErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if r.Body == nil {
			return fmt.Errorf("early nil body")
		}
		data, err := handler(r.Context(), r)
		if err == nil {
			w.Header().Add("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(data)
		}
		return err
	}
}

func expectsJSON[Expects, Returns any](handler handlerFunc[Expects, Returns]) httpTypeFunc[Returns] {
	return func(ctx context.Context, req *http.Request) (ret Returns, err error) {
		if req.Body == nil {
			return ret, fmt.Errorf("weird late nil body")
		}
		data, err := GetRequestBody[Expects](req)
		if err != nil {
			return ret, err
		}
		return handler(ctx, *data)
	}
}

func expectsNothing[Returns any](handler func(context.Context) (Returns, error)) httpTypeFunc[Returns] {
	return func(ctx context.Context, r *http.Request) (Returns, error) {
		return handler(ctx)
	}
}

func requiresLogin(api *model.ModelAPI, secret string, handler httpErrorFunc) httpErrorFunc {
	return func(w http.ResponseWriter, req *http.Request) error {
		user, err := getUserFromRequest(req.Context(), api, req.Header.Get("Authorization"), secret)
		if err != nil {
			return err
		} else if req.Body == nil {
			return fmt.Errorf("super early nil body")
		} else if user == nil {
			return fmt.Errorf("no user supplied")
		}
		ctx := context.WithValue(req.Context(), userContextKey{}, user)
		return handler(w, req.WithContext(ctx))
	}
}

func generateJWT(
	secret string,
	username string,
) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	//claims["exp"] = time.Now().Add(24 * time.Hour)
	claims["authorized"] = true
	claims["user"] = username
	return token.SignedString([]byte(secret))
}

func parseJWT(
	secret string,
	tokenString string,
) (string, error) {
	signingKey := []byte(secret)
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("could not parse claims")
	}
	return claims["user"].(string), nil
}

func getUserFromRequest(
	ctx context.Context,
	model *model.ModelAPI,
	authHeader string,
	secret string,
) (*types.User, error) {
	// extract the JWT token from the bearer string
	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	if tokenString == "" {
		return nil, fmt.Errorf("no token provided")
	}
	username, err := parseJWT(secret, tokenString)
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %s", err.Error())
	}
	user, err := model.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %s", err.Error())
	}
	user.HashedPassword = ""
	return user, nil
}
