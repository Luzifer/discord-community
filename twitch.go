package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/v2/backoff"
	"github.com/pkg/errors"
)

const (
	twitchAPIRequestLimit   = 5
	twitchAPIRequestTimeout = 2 * time.Second
)

type (
	twitchAdapter struct {
		clientID     string
		clientSecret string
		token        string
	}

	twitchStreamListing struct {
		Data []struct {
			ID           string    `json:"id"`
			UserID       string    `json:"user_id"`
			UserLogin    string    `json:"user_login"`
			UserName     string    `json:"user_name"`
			GameID       string    `json:"game_id"`
			GameName     string    `json:"game_name"`
			Type         string    `json:"type"`
			Title        string    `json:"title"`
			ViewerCount  int64     `json:"viewer_count"`
			StartedAt    time.Time `json:"started_at"`
			Language     string    `json:"language"`
			ThumbnailURL string    `json:"thumbnail_url"`
			TagIds       []string  `json:"tag_ids"`
			IsMature     bool      `json:"is_mature"`
		} `json:"data"`
		Pagination struct {
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}

	twitchStreamSchedule struct {
		Data struct {
			Segments []struct {
				ID            string     `json:"id"`
				StartTime     *time.Time `json:"start_time"`
				EndTime       *time.Time `json:"end_time"`
				Title         string     `json:"title"`
				CanceledUntil *time.Time `json:"canceled_until"`
				Category      *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
				IsRecurring bool `json:"is_recurring"`
			} `json:"segments"`
			BroadcasterID    string `json:"broadcaster_id"`
			BroadcasterName  string `json:"broadcaster_name"`
			BroadcasterLogin string `json:"broadcaster_login"`
			Vacation         *struct {
				StartTime *time.Time `json:"start_time"`
				EndTime   *time.Time `json:"end_time"`
			} `json:"vacation"`
		} `json:"data"`
		Pagination struct {
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}
)

func newTwitchAdapter(clientID, clientSecret, token string) *twitchAdapter {
	return &twitchAdapter{
		clientID:     clientID,
		clientSecret: clientSecret,
		token:        token,
	}
}

func (t twitchAdapter) GetChannelStreamSchedule(ctx context.Context, broadcasterID string, startTime *time.Time) (*twitchStreamSchedule, error) {
	out := &twitchStreamSchedule{}

	params := make(url.Values)
	params.Set("broadcaster_id", broadcasterID)
	if startTime != nil {
		params.Set("start_time", startTime.Format(time.RFC3339))
	}

	return out, backoff.NewBackoff().
		WithMaxIterations(twitchAPIRequestLimit).
		Retry(func() error {
			return errors.Wrap(
				t.request(ctx, http.MethodGet, "/helix/schedule", params, nil, out),
				"fetching schedule",
			)
		})
}

func (t twitchAdapter) GetStreamsForUser(ctx context.Context, userName string) (*twitchStreamListing, error) {
	out := &twitchStreamListing{}

	params := make(url.Values)
	params.Set("user_login", strings.ToLower(userName))

	return out, backoff.NewBackoff().
		WithMaxIterations(twitchAPIRequestLimit).
		Retry(func() error {
			return errors.Wrap(
				t.request(ctx, http.MethodGet, "/helix/streams", params, nil, out),
				"fetching streams",
			)
		})
}

func (t twitchAdapter) getAppAccessToken(ctx context.Context) (string, error) {
	var rData struct {
		AccessToken  string        `json:"access_token"`
		RefreshToken string        `json:"refresh_token"`
		ExpiresIn    int           `json:"expires_in"`
		Scope        []interface{} `json:"scope"`
		TokenType    string        `json:"token_type"`
	}

	params := make(url.Values)
	params.Set("client_id", t.clientID)
	params.Set("client_secret", t.clientSecret)
	params.Set("grant_type", "client_credentials")

	u, _ := url.Parse("https://id.twitch.tv/oauth2/token")
	u.RawQuery = params.Encode()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "fetching response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", errors.Wrapf(err, "unexpected status %d and cannot read body", resp.StatusCode)
		}
		return "", errors.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	return rData.AccessToken, errors.Wrap(
		json.NewDecoder(resp.Body).Decode(&rData),
		"decoding response",
	)
}

func (t twitchAdapter) request(ctx context.Context, method, path string, params url.Values, body io.Reader, output interface{}) error {
	ctxTimed, cancel := context.WithTimeout(ctx, twitchAPIRequestTimeout)
	defer cancel()

	u, _ := url.Parse(strings.Join([]string{
		"https://api.twitch.tv",
		strings.TrimLeft(path, "/"),
	}, "/"))

	if params != nil {
		u.RawQuery = params.Encode()
	}

	req, _ := http.NewRequestWithContext(ctxTimed, method, u.String(), body)
	req.Header.Set("Authorization", strings.Join([]string{"Bearer", t.token}, " "))
	req.Header.Set("Client-Id", t.clientID)

	if t.token == "" {
		accessToken, err := t.getAppAccessToken(ctx)
		if err != nil {
			return errors.Wrap(err, "fetching app-access-token")
		}
		req.Header.Set("Authorization", strings.Join([]string{"Bearer", accessToken}, " "))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "fetching response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "unexpected status %d and cannot read body", resp.StatusCode)
		}
		return errors.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	if output == nil {
		return nil
	}

	return errors.Wrap(
		json.NewDecoder(resp.Body).Decode(output),
		"decoding response",
	)
}
