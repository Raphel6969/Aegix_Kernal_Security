package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/Raphel6969/Kernal_AI_Security/backend/internal/auth"
	"github.com/Raphel6969/Kernal_AI_Security/backend/internal/model"
)

var (
	oauthStateCookieName = "oauthstate"
)

func getOAuthConfig(r *http.Request, provider string) (*oauth2.Config, error) {
	// Dynamically determine callback URL based on environment or request headers
	callbackURL := ""
	if base := os.Getenv("RENDER_EXTERNAL_URL"); base != "" {
		callbackURL = strings.TrimSuffix(base, "/") + "/api/auth/" + provider + "/callback"
	} else if base := os.Getenv("BACKEND_URL"); base != "" && !strings.Contains(base, "localhost:8000") {
		callbackURL = strings.TrimSuffix(base, "/") + "/api/auth/" + provider + "/callback"
	} else if r != nil {
		proto := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			proto = "https"
		}
		host := r.Host
		if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
			host = xfh
		}
		callbackURL = fmt.Sprintf("%s://%s/api/auth/%s/callback", proto, host, provider)
	} else {
		callbackURL = "http://localhost:8000/api/auth/" + provider + "/callback"
	}

	switch provider {
	case "google":
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("Google OAuth not configured: GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required")
		}
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		}, nil
	case "github":
		clientID := os.Getenv("GITHUB_CLIENT_ID")
		clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
		if clientID == "" || clientSecret == "" {
			return nil, fmt.Errorf("GitHub OAuth not configured: GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET are required")
		}
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("provider %s not supported", provider)
	}
}

func generateStateOauthCookie(w http.ResponseWriter, isHTTPS bool) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteLaxMode, // Lax required for cross-site OAuth redirects
		Path:     "/",
	}
	http.SetCookie(w, &cookie)
	return state
}

func (s *Server) handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	conf, err := getOAuthConfig(r, provider)
	if err != nil {
		slog.Warn("OAuth login failed", "provider", provider, "err", err)
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	state := generateStateOauthCookie(w, isHTTPS)
	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Validate state
	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || r.FormValue("state") != stateCookie.Value {
		writeErr(w, http.StatusBadRequest, "invalid oauth state")
		return
	}

	conf, err := getOAuthConfig(r, provider)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	code := r.FormValue("code")
	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		slog.Error("OAuth token exchange failed", "provider", provider, "err", err)
		writeErr(w, http.StatusInternalServerError, "failed to exchange token")
		return
	}

	// Fetch user info based on provider
	var email, providerID string
	client := conf.Client(context.Background(), token)

	if provider == "google" {
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to get google user info")
			return
		}
		defer resp.Body.Close()

		var gUser struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to decode google user info")
			return
		}
		email = gUser.Email
		providerID = gUser.ID
	} else if provider == "github" {
		resp, err := client.Get("https://api.github.com/user")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to get github user info")
			return
		}
		defer resp.Body.Close()

		var ghUser struct {
			ID    int64  `json:"id"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to decode github user info")
			return
		}

		providerID = fmt.Sprintf("%d", ghUser.ID)
		email = ghUser.Email

		// If email is private in GitHub profile, fetch from emails endpoint
		if email == "" {
			emailResp, err := client.Get("https://api.github.com/user/emails")
			if err == nil {
				defer emailResp.Body.Close()
				var emails []struct {
					Email    string `json:"email"`
					Primary  bool   `json:"primary"`
					Verified bool   `json:"verified"`
				}
				if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
					for _, e := range emails {
						if e.Primary && e.Verified {
							email = e.Email
							break
						}
					}
					if email == "" && len(emails) > 0 {
						email = emails[0].Email
					}
				}
			}
		}
	}

	if email == "" {
		writeErr(w, http.StatusBadRequest, "unable to retrieve verified email from provider")
		return
	}

	// Upsert User in DB
	user := &model.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: "", // No password for OAuth
		Provider:     provider,
		ProviderID:   providerID,
		Role:         model.RoleViewer, // Default role
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if s.svc.DB != nil {
		if err := s.svc.DB.UpsertOAuthUser(r.Context(), user); err != nil {
			slog.Error("Failed to upsert OAuth user", "error", err)
			writeErr(w, http.StatusInternalServerError, "failed to save user")
			return
		}
	}

	// Generate JWT Tokens
	tokens, err := auth.GenerateTokens(user)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to generate tokens")
		return
	}

	isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	// Set refresh token in HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	// Redirect to frontend root where the auth interceptor will automatically refresh token
	redirectURL := "/"
	if len(s.cfg.FrontendOrigins) > 0 {
		first := s.cfg.FrontendOrigins[0]
		if first != "*" && (strings.HasPrefix(first, "http://") || strings.HasPrefix(first, "https://")) {
			redirectURL = first
		}
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}
