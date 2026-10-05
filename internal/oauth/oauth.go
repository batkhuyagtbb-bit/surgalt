// Package oauth нь Google, Microsoft (OpenID Connect), Facebook, Instagram
// (OAuth2)-аар нэвтрэх үйлчилгээ үзүүлэгчдийг тодорхойлно.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Identity нь үйлчилгээ үзүүлэгчээс ирсэн хэрэглэгчийн мэдээлэл.
type Identity struct {
	Provider      string
	Subject       string // үйлчилгээ үзүүлэгч доторх давтагдашгүй ID
	Email         string
	EmailVerified bool
	Name          string
	Username      string
	AvatarURL     string
}

type Provider struct {
	Name   string
	Label  string
	Config oauth2.Config
	PKCE   bool
	fetch  func(ctx context.Context, hc *http.Client, tok *oauth2.Token) (Identity, error)
}

func (p *Provider) FetchIdentity(ctx context.Context, tok *oauth2.Token) (Identity, error) {
	hc := p.Config.Client(ctx, tok)
	hc.Timeout = 10 * time.Second
	id, err := p.fetch(ctx, hc, tok)
	id.Provider = p.Name
	if err == nil && id.Subject == "" {
		err = fmt.Errorf("%s: хэрэглэгчийн ID ирсэнгүй", p.Name)
	}
	return id, err
}

type Credentials struct{ ClientID, ClientSecret string }

// Registry нь тохируулагдсан (client ID-тай) үйлчилгээ үзүүлэгчдийг агуулна.
type Registry struct {
	byName map[string]*Provider
	order  []*Provider
}

func (r *Registry) Get(name string) (*Provider, bool) {
	p, ok := r.byName[name]
	return p, ok
}

func (r *Registry) List() []*Provider { return r.order }

// NewRegistry: baseURL нь callback хаягийн эх (https://surgalt.mn).
// Client ID хоосон үйлчилгээ үзүүлэгч идэвхгүй.
func NewRegistry(baseURL string, google, microsoft, facebook, instagram Credentials) *Registry {
	r := &Registry{byName: map[string]*Provider{}}
	cb := func(n string) string { return strings.TrimRight(baseURL, "/") + "/auth/" + n + "/callback" }
	add := func(p *Provider, c Credentials) {
		if c.ClientID == "" {
			return
		}
		p.Config.ClientID, p.Config.ClientSecret, p.Config.RedirectURL = c.ClientID, c.ClientSecret, cb(p.Name)
		r.byName[p.Name] = p
		r.order = append(r.order, p)
	}

	add(&Provider{
		Name: "google", Label: "Google (Gmail)", PKCE: true,
		Config: oauth2.Config{
			Scopes: []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		fetch: oidcUserinfo("https://openidconnect.googleapis.com/v1/userinfo", true),
	}, google)

	add(&Provider{
		Name: "microsoft", Label: "Microsoft (Outlook, Hotmail)", PKCE: true,
		Config: oauth2.Config{
			Scopes: []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		// Microsoft-ын email claim-ийг баталгаажсан гэж үзэхгүй (аль ч tenant тохируулж болно).
		fetch: oidcUserinfo("https://graph.microsoft.com/oidc/userinfo", false),
	}, microsoft)

	add(&Provider{
		Name: "facebook", Label: "Facebook",
		Config: oauth2.Config{
			Scopes: []string{"public_profile", "email"},
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://www.facebook.com/v21.0/dialog/oauth", TokenURL: "https://graph.facebook.com/v21.0/oauth/access_token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		fetch: facebookUser,
	}, facebook)

	add(&Provider{
		Name: "instagram", Label: "Instagram",
		Config: oauth2.Config{
			// Instagram API with Instagram Login (мэргэжлийн/creator данс).
			Scopes: []string{"instagram_business_basic"},
			Endpoint: oauth2.Endpoint{
				AuthURL: "https://www.instagram.com/oauth/authorize", TokenURL: "https://api.instagram.com/oauth/access_token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		fetch: instagramUser,
	}, instagram)

	return r
}

func getJSON(ctx context.Context, hc *http.Client, u string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("userinfo %s: %d %s", u, res.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, dst)
}

func oidcUserinfo(endpoint string, trustEmailVerified bool) func(context.Context, *http.Client, *oauth2.Token) (Identity, error) {
	return func(ctx context.Context, hc *http.Client, _ *oauth2.Token) (Identity, error) {
		var v struct {
			Sub           string `json:"sub"`
			Email         string `json:"email"`
			EmailVerified any    `json:"email_verified"` // bool эсвэл "true"
			Name          string `json:"name"`
			Picture       string `json:"picture"`
		}
		if err := getJSON(ctx, hc, endpoint, &v); err != nil {
			return Identity{}, err
		}
		verified := v.EmailVerified == true || v.EmailVerified == "true"
		return Identity{Subject: v.Sub, Email: strings.ToLower(v.Email), EmailVerified: trustEmailVerified && verified,
			Name: v.Name, AvatarURL: v.Picture}, nil
	}
}

func facebookUser(ctx context.Context, hc *http.Client, _ *oauth2.Token) (Identity, error) {
	var v struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	err := getJSON(ctx, hc, "https://graph.facebook.com/v21.0/me?fields="+url.QueryEscape("id,name,email,picture.type(large)"), &v)
	return Identity{Subject: v.ID, Name: v.Name, Email: strings.ToLower(v.Email), AvatarURL: v.Picture.Data.URL}, err
}

func instagramUser(ctx context.Context, hc *http.Client, tok *oauth2.Token) (Identity, error) {
	var v struct {
		UserID   string `json:"user_id"`
		ID       string `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
		Picture  string `json:"profile_picture_url"`
	}
	u := "https://graph.instagram.com/v21.0/me?fields=user_id,username,name,profile_picture_url&access_token=" + url.QueryEscape(tok.AccessToken)
	err := getJSON(ctx, http.DefaultClient, u, &v)
	sub := v.UserID
	if sub == "" {
		sub = v.ID
	}
	name := v.Name
	if name == "" {
		name = v.Username
	}
	return Identity{Subject: sub, Username: v.Username, Name: name, AvatarURL: v.Picture}, err
}
