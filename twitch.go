package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Luzifer/go_helpers/v2/backoff"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
			TagIDs       []string  `json:"tag_ids"`
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

	twitchUserListing struct {
		Data []struct {
			ID              string    `json:"id"`
			Login           string    `json:"login"`
			DisplayName     string    `json:"display_name"`
			Type            string    `json:"type"`
			BroadcasterType string    `json:"broadcaster_type"`
			Description     string    `json:"description"`
			ProfileImageURL string    `json:"profile_image_url"`
			OfflineImageURL string    `json:"offline_image_url"`
			ViewCount       int64     `json:"view_count"`
			Email           string    `json:"email"`
			CreatedAt       time.Time `json:"created_at"`
		} `json:"data"`
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

	if err := backoff.NewBackoff().
		WithMaxIterations(twitchAPIRequestLimit).
		Retry(func() error {
			return errors.Wrap(
				t.request(ctx, http.MethodGet, "/helix/schedule", params, nil, out),
				"fetching schedule",
			)
		}); err != nil {
		return nil, fmt.Errorf("getting schedule: %w", err)
	}

	return out, nil
}

func (t twitchAdapter) GetStreamsForUser(ctx context.Context, userNames ...string) (*twitchStreamListing, error) {
	out := &twitchStreamListing{}

	params := make(url.Values)
	params.Set("first", "100")
	params["user_login"] = userNames

	if err := backoff.NewBackoff().
		WithMaxIterations(twitchAPIRequestLimit).
		Retry(func() error {
			return errors.Wrap(
				t.request(ctx, http.MethodGet, "/helix/streams", params, nil, out),
				"fetching streams",
			)
		}); err != nil {
		return nil, fmt.Errorf("getting streams: %w", err)
	}

	return out, nil
}

func (t twitchAdapter) GetUserByUsername(ctx context.Context, userNames ...string) (*twitchUserListing, error) {
	out := &twitchUserListing{}

	params := make(url.Values)
	params.Set("first", "100")
	params["login"] = userNames

	if err := backoff.NewBackoff().
		WithMaxIterations(twitchAPIRequestLimit).
		Retry(func() error {
			return errors.Wrap(
				t.request(ctx, http.MethodGet, "/helix/users", params, nil, out),
				"fetching user",
			)
		}); err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return out, nil
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("closing Twitch response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logrus.WithError(err).Error("closing Twitch response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
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
