package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/model"
	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/types"
	"github.com/golang-jwt/jwt"
)

func GetRequestBody[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	var requestBody T

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r.Body)
	if err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}
	err = json.Unmarshal(buf.Bytes(), &requestBody)
	if err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}
	return &requestBody, nil
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
	model *model.ModelAPI,
	req *http.Request,
	secret string,
) (*types.User, error) {
	// extract the JWT token from the bearer string
	tokenString := strings.Replace(req.Header.Get("Authorization"), "Bearer ", "", 1)
	if tokenString == "" {
		return nil, fmt.Errorf("no token provided")
	}
	username, err := parseJWT(secret, tokenString)
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %s", err.Error())
	}
	user, err := model.GetUser(context.Background(), username)
	if err != nil {
		return nil, fmt.Errorf("error parsing token: %s", err.Error())
	}
	user.HashedPassword = ""
	return user, nil
}
