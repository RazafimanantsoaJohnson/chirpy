package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	authorization := headers.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("no authorization was provided")
	}
	return strings.Replace(authorization, "Bearer ", "", 1), nil
}

func GetApiKey(header http.Header) (string, error) { // used to get the 'API key' of 'Polka webhook client'
	authorization := header.Get("Authorization")
	if authorization == "" {
		return "", fmt.Errorf("no authorization was provided")
	}
	return strings.Replace(authorization, "ApiKey ", "", 1), nil
}
