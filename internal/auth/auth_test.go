package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateAndValidateJWT(t *testing.T) {
	cases := []struct {
		id          uuid.UUID
		tokenSecret string
	}{
		{
			id:          uuid.New(),
			tokenSecret: "secret1",
		},
		{
			id:          uuid.New(),
			tokenSecret: "secret2",
		},
	}
	for _, c := range cases {
		token, err := MakeJWT(c.id, c.tokenSecret, 5*time.Minute)
		if err != nil {
			fmt.Println(err)
			t.Errorf("An error occured")
			return
		}
		tokenUserId, err := ValidateJWT(token, c.tokenSecret)
		if err != nil {
			fmt.Println(err)
			t.Errorf("An error occured when validating the JWT")
			return
		}
		falseToken, _ := ValidateJWT(token, "a false secret")
		if tokenUserId.String() != c.id.String() {
			t.Errorf("the unsigned token %v, is different from the given token: %v", tokenUserId.String(), c.id.String())
		}
		if falseToken.String() == c.id.String() {
			t.Errorf("the token created with a wrong secret was 'validated':\n\t %v should be different to: %v", falseToken.String(), c.id.String())
		}
	}

}

func TestTokenExpiry(t *testing.T) {
	// test token expiry
	token, _ := MakeJWT(uuid.New(), "secret key", 2*time.Second)
	time.Sleep(3 * time.Second)
	_, err := ValidateJWT(token, "secret key")
	if err == nil {
		t.Errorf("The token should not be validated, it should be expired")
	}
}

func TestGetRequestToken(t *testing.T) {
	cases := []string{
		"token1", "token2",
	}
	for _, c := range cases {
		req, err := http.NewRequest("GET", "http://nowhere.com", nil)
		req.Header.Add("Authorization", "Bearer "+c)
		token, err := GetBearerToken(req.Header)
		if err != nil {
			t.Errorf("error happen when getting the bearer token")
			return
		}
		if c != token {
			t.Errorf("error: provided token '%v' != received token '%v'", c, token)
			return
		}
	}
}

func TestCreateRefreshToken(t *testing.T) {
	_, err := MakeRefreshToken()
	if err != nil {
		t.Errorf("error happened when generating token %v", err)
		return
	}
	// t.Errorf("the new Random: %v", newRandom)
}
