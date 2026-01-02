package auth

import (
	"context"
	"log"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
)

var Verifier *oidc.IDTokenVerifier

func InitKeycloak() {
	issuer := os.Getenv("KEYCLOAK_ISSUER")
	clientID := os.Getenv("KEYCLOAK_CLIENT_ID")

	if issuer == "" || clientID == "" {
		log.Fatal("KEYCLOAK_ISSUER or KEYCLOAK_CLIENT_ID not set")
	}

	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		log.Fatalf("Keycloak provider error: %v", err)
	}

	Verifier = provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	log.Println("Keycloak OIDC initialized")
}
