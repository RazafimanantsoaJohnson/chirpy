package auth

import (
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJwt(t *testing.T) {
	cases := []string{"string1", "string3"}
	for _, c := range cases {
		log.Println(c)
		token, err := MakeJWT(uuid.New(), "heheboy", 5*time.Minute)
		if err != nil {
			t.Errorf(err.Error())
		}
		t.Logf(token)
	}
}
