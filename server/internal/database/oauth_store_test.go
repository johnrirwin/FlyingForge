package database

import (
	"context"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

func TestOAuthStoreCleanupExpiredOAuthState(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	t.Cleanup(func() { testDB.Close() })
	t.Cleanup(func() { testDB.Cleanup(context.Background()) })

	db := &DB{DB: testDB.DB}
	userStore := NewUserStore(db)
	oauthStore := NewOAuthStore(db)
	ctx := context.Background()

	user, err := userStore.Create(ctx, models.CreateUserParams{
		Email:  "pilot@example.com",
		Status: models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	client, err := oauthStore.CreateClient(
		ctx,
		"ff_mcp_cleanup_test",
		"Cleanup Test Connector",
		[]string{"https://chat.openai.com/a/oauth/callback"},
		[]string{models.OAuthGrantTypeAuthorizationCode, models.OAuthGrantTypeRefreshToken},
		[]string{models.OAuthResponseTypeCode},
		models.OAuthTokenEndpointAuthMethodNone,
		"flyingforge.read",
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if _, err := oauthStore.CreateAuthorizationCode(
		ctx,
		"expired-auth-code",
		client.ClientID,
		user.ID,
		client.RedirectURIs[0],
		"flyingforge.read",
		"https://flyingforge.example/mcp",
		"challenge",
		models.OAuthCodeChallengeMethodS256,
		time.Now().Add(-time.Minute),
	); err != nil {
		t.Fatalf("create expired authorization code: %v", err)
	}

	liveCode, err := oauthStore.CreateAuthorizationCode(
		ctx,
		"live-auth-code",
		client.ClientID,
		user.ID,
		client.RedirectURIs[0],
		"flyingforge.read",
		"https://flyingforge.example/mcp",
		"challenge",
		models.OAuthCodeChallengeMethodS256,
		time.Now().Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("create live authorization code: %v", err)
	}
	if _, err := oauthStore.ConsumeAuthorizationCode(ctx, "live-auth-code"); err != nil {
		t.Fatalf("consume live authorization code: %v", err)
	}
	if liveCode == nil {
		t.Fatal("expected consumed authorization code record")
	}

	if _, err := oauthStore.CreateRefreshToken(
		ctx,
		user.ID,
		client.ClientID,
		"expired-refresh-token",
		"flyingforge.read",
		"https://flyingforge.example/mcp",
		time.Now().Add(-time.Minute),
	); err != nil {
		t.Fatalf("create expired refresh token: %v", err)
	}

	if _, err := oauthStore.CreateRefreshToken(
		ctx,
		user.ID,
		client.ClientID,
		"revoked-refresh-token",
		"flyingforge.read",
		"https://flyingforge.example/mcp",
		time.Now().Add(time.Hour),
	); err != nil {
		t.Fatalf("create active refresh token: %v", err)
	}
	if _, err := oauthStore.ConsumeRefreshToken(ctx, "revoked-refresh-token"); err != nil {
		t.Fatalf("consume refresh token: %v", err)
	}

	if err := oauthStore.CleanupExpiredOAuthState(ctx); err != nil {
		t.Fatalf("cleanup OAuth state: %v", err)
	}

	assertRowCount(t, testDB, "oauth_authorization_codes", 0)
	assertRowCount(t, testDB, "oauth_refresh_tokens", 0)
}

func assertRowCount(t *testing.T, testDB *testutil.TestDB, table string, want int) {
	t.Helper()

	var got int
	if err := testDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("expected %d rows in %s, got %d", want, table, got)
	}
}
