package apiserver

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Saaghh/wallet/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func (s *APIServer) JWTAuth(next http.Handler) http.Handler {
	var fn http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		claims, err := getClaimsFromHeader(r.Header.Get("Authorization"), s.key)

		switch {
		case errors.Is(err, model.ErrInvalidAccessToken):
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized")

			return
		case err != nil:
			zap.L().With(zap.Error(err)).Warn("JWTAuth/getClaimsFromHeader(r.Header.Get(\"Authorization\"), s.key)")
			writeErrorResponse(w, http.StatusInternalServerError, "internal server error")

			return
		}

		expiresAtTime := time.Unix(claims.ExpiresAt.Unix(), 0)
		if expiresAtTime.Before(time.Now()) {
			writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized")

			return
		}

		userInfo := model.UserInfo{
			ID: claims.UUID,
		}

		r = r.WithContext(context.WithValue(r.Context(), model.UserInfoKey, userInfo))
		next.ServeHTTP(w, r)
	}

	return fn
}

func parseToken(accessToken string, key *rsa.PublicKey) (*model.Claims, error) {
	token, err := jwt.ParseWithClaims(accessToken, &model.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, model.ErrInvalidAccessToken
		}

		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt.ParseWithClaims(...): %w", err)
	}

	claims, ok := token.Claims.(*model.Claims)
	if !(ok && token.Valid) {
		return nil, model.ErrInvalidAccessToken
	}

	return claims, nil
}

func getClaimsFromHeader(authHeader string, key *rsa.PublicKey) (*model.Claims, error) {
	headerParts := strings.Split(authHeader, " ")

	switch {
	case authHeader == "":
		fallthrough
	case len(headerParts) != 2:
		fallthrough
	case headerParts[0] != "Bearer":
		return nil, model.ErrInvalidAccessToken
	}

	claims, err := parseToken(headerParts[1], key)
	if err != nil {
		return nil, fmt.Errorf("parseToken(headerParts[1], key): %w", err)
	}

	return claims, nil
}

func (s *APIServer) Metrics(next http.Handler) http.Handler {
	var fn http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		defer s.metrics.TrackHTTPRequest(time.Now(), r)

		next.ServeHTTP(w, r)
	}

	return fn
}
