package tokens

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

// TokenContent represents the content we store in a JWT token - the username and k8s token
type TokenContent struct {
	Username string
	Token    string
}

// CreateJWTToken creates a jwt token containing a username and another token using the input secret
func CreateJWTToken(username string, token string, secret []byte) (string, error) {
	result := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"token":    token,
	})

	// Sign and get the complete encoded token as a string using the secret
	return result.SignedString(secret)
}

// ParseJWTToken parses the token string into a TokenContent
func ParseJWTToken(tokenString string, secret []byte) (content *TokenContent, err error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &TokenContent{
			Username: claims["username"].(string),
			Token:    claims["token"].(string),
		}, nil
	}

	return nil, fmt.Errorf("Unknown error getting token, claim or token is not ok")
}
