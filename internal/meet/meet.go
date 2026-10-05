// Package meet нь багшийн Google Calendar-т Google Meet бүхий уулзалт автоматаар үүсгэнэ.
package meet

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

var ErrNotConnected = errors.New("Google Meet холбогдоогүй эсвэл эрх цуцлагдсан")

const calendarScope = "https://www.googleapis.com/auth/calendar.events"

type Client struct {
	cfg oauth2.Config
	api string // тестэд солих боломжтой
}

// New: Google OAuth client (нэвтрэлтийнхтэй ижил) — redirect нь /auth/google-meet/callback.
func New(baseURL, clientID, clientSecret string) *Client {
	if clientID == "" {
		return nil
	}
	return &Client{
		cfg: oauth2.Config{
			ClientID: clientID, ClientSecret: clientSecret,
			RedirectURL: strings.TrimRight(baseURL, "/") + "/auth/google-meet/callback",
			Scopes:      []string{"openid", "email", calendarScope},
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		api: "https://www.googleapis.com/calendar/v3",
	}
}

// AuthURL: offline + consent — refresh token авахын тулд.
func (c *Client) AuthURL(state, verifier string) string {
	return c.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce, oauth2.S256ChallengeOption(verifier))
}

// Exchange нь refresh token буцаана.
func (c *Client) Exchange(ctx context.Context, code, verifier string) (string, error) {
	tok, err := c.cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", err
	}
	if tok.RefreshToken == "" {
		return "", errors.New("Google refresh token өгсөнгүй — дахин холбоно уу")
	}
	return tok.RefreshToken, nil
}

type Meeting struct {
	EventID string
	MeetURL string
}

// Create нь багшийн primary календарт Meet-тэй event үүсгэнэ.
func (c *Client) Create(ctx context.Context, refreshToken, title, description string, start time.Time, dur time.Duration) (*Meeting, error) {
	hc := c.cfg.Client(ctx, &oauth2.Token{RefreshToken: refreshToken})
	hc.Timeout = 15 * time.Second
	var rid [8]byte
	_, _ = rand.Read(rid[:])
	body, _ := json.Marshal(map[string]any{
		"summary":     title,
		"description": description,
		"start":       map[string]string{"dateTime": start.UTC().Format(time.RFC3339)},
		"end":         map[string]string{"dateTime": start.Add(dur).UTC().Format(time.RFC3339)},
		"conferenceData": map[string]any{"createRequest": map[string]any{
			"requestId":             hex.EncodeToString(rid[:]),
			"conferenceSolutionKey": map[string]string{"type": "hangoutsMeet"},
		}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api+"/calendars/primary/events?conferenceDataVersion=1", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := hc.Do(req)
	if err != nil {
		var re *oauth2.RetrieveError
		if errors.As(err, &re) {
			return nil, ErrNotConnected
		}
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return nil, ErrNotConnected
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("calendar: %d %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	var ev struct {
		ID          string `json:"id"`
		HangoutLink string `json:"hangoutLink"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		return nil, err
	}
	if ev.HangoutLink == "" {
		return nil, errors.New("Google Meet холбоос үүссэнгүй (Workspace тохиргоог шалгана уу)")
	}
	return &Meeting{EventID: ev.ID, MeetURL: ev.HangoutLink}, nil
}
