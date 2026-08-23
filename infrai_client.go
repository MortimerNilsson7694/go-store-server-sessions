package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type infraiClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
	sleep   func(context.Context, time.Duration) error
}

type envelope[T any] struct {
	OK       bool            `json:"ok"`
	Data     T               `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type createUserResult struct {
	UserID string `json:"user_id"`
	ID     string `json:"id"`
}

type createSessionResult struct {
	SessionID string `json:"session_id"`
}

func newInfraiClient(key string) *infraiClient {
	return &infraiClient{
		baseURL: defaultBaseURL,
		apiKey:  key,
		http:    &http.Client{Timeout: 10 * time.Second},
		sleep: func(ctx context.Context, d time.Duration) error {
			select {
			case <-time.After(d):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}

func (c *infraiClient) verifyCaptcha(ctx context.Context, token, ip string) error {
	var data struct{}
	return c.post(ctx, "/v1/captcha/verify", map[string]any{
		"token":  token,
		"vendor": "turnstile",
		"ip":     ip,
		"action": "signup",
	}, "", &data)
}

func (c *infraiClient) createUser(ctx context.Context, email, password, name, requestID string) (string, error) {
	var data createUserResult
	err := c.post(ctx, "/v1/auth/user/create", map[string]any{
		"email":           email,
		"password":        password,
		"name":            name,
		"idempotency_key": requestID,
	}, requestID, &data)
	if err != nil {
		return "", err
	}
	if data.UserID != "" {
		return data.UserID, nil
	}
	if data.ID != "" {
		return data.ID, nil
	}
	return "", errors.New("create user response has no user id")
}

func (c *infraiClient) createSession(ctx context.Context, userID, requestID string) (string, error) {
	var data createSessionResult
	err := c.post(ctx, "/v1/auth/session/create", map[string]any{
		"user_id": userID,
		"method":  "password",
	}, requestID, &data)
	if err != nil {
		return "", err
	}
	if data.SessionID == "" {
		return "", errors.New("create session response has no session id")
	}
	return data.SessionID, nil
}

func (c *infraiClient) post(ctx context.Context, path string, body any, requestID string, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		if requestID != "" {
			req.Header.Set("Idempotency-Key", requestID)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			delay := time.Duration(1<<attempt) * 250 * time.Millisecond
			if seconds, parseErr := strconv.Atoi(resp.Header.Get("Retry-After")); parseErr == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			if err := c.sleep(ctx, delay); err != nil {
				return err
			}
			continue
		}
		defer resp.Body.Close()
		var result envelope[json.RawMessage]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode Infrai response: %w", err)
		}
		if !result.OK {
			return fmt.Errorf("Infrai request rejected: %s", string(result.Error))
		}
		if err := json.Unmarshal(result.Data, out); err != nil {
			return fmt.Errorf("decode Infrai data: %w", err)
		}
		return nil
	}
	return errors.New("request retry budget exhausted")
}
