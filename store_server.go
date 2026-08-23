package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type authGateway interface {
	verifyCaptcha(ctx context.Context, token, ip string) error
	createUser(ctx context.Context, email, password, name, requestID string) (string, error)
	createSession(ctx context.Context, userID, requestID string) (string, error)
}

type storeServer struct {
	auth     authGateway
	orders   *orderBook
	mu       sync.Mutex
	users    map[string]string
	sessions map[string]string
	now      func() time.Time
}

type signupInput struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	CaptchaToken string `json:"captcha_token"`
	RequestID    string `json:"request_id"`
}

type loginInput struct {
	Email     string `json:"email"`
	RequestID string `json:"request_id"`
}

type checkoutInput struct {
	OrderID  string `json:"order_id"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

func newStoreServer(auth authGateway) *storeServer {
	return &storeServer{auth: auth, orders: newOrderBook(), users: make(map[string]string), sessions: make(map[string]string), now: time.Now}
}

func (s *storeServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /signup", s.signup)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /checkout", s.withUser(s.checkout))
	mux.HandleFunc("POST /orders/{id}/fulfill", s.withUser(s.fulfill))
	mux.HandleFunc("GET /orders", s.withUser(s.ordersForCustomer))
	return mux
}

func (s *storeServer) signup(w http.ResponseWriter, r *http.Request) {
	var in signupInput
	if !decode(w, r, &in) {
		return
	}
	if in.Email == "" || in.Password == "" || in.CaptchaToken == "" || in.RequestID == "" {
		writeError(w, http.StatusBadRequest, "email, password, captcha_token, and request_id are required")
		return
	}
	if err := s.auth.verifyCaptcha(r.Context(), in.CaptchaToken, r.RemoteAddr); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	userID, err := s.auth.createUser(r.Context(), in.Email, in.Password, in.Name, in.RequestID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.mu.Lock()
	s.users[in.Email] = userID
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]string{"user_id": userID})
}

func (s *storeServer) login(w http.ResponseWriter, r *http.Request) {
	var in loginInput
	if !decode(w, r, &in) {
		return
	}
	s.mu.Lock()
	userID := s.users[in.Email]
	s.mu.Unlock()
	if userID == "" || in.RequestID == "" {
		writeError(w, http.StatusUnauthorized, "invalid login")
		return
	}
	upstreamID, err := s.auth.createSession(r.Context(), userID, in.RequestID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	localID, err := randomID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}
	s.mu.Lock()
	s.sessions[localID] = userID + ":" + upstreamID
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "store_session", Value: localID, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: true})
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed_in"})
}

func (s *storeServer) withUser(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("store_session")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}
		s.mu.Lock()
		stored := s.sessions[cookie.Value]
		s.mu.Unlock()
		userID := stored
		for i, ch := range stored {
			if ch == ':' {
				userID = stored[:i]
				break
			}
		}
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "login required")
			return
		}
		next(w, r, userID)
	}
}

func (s *storeServer) checkout(w http.ResponseWriter, r *http.Request, userID string) {
	var in checkoutInput
	if !decode(w, r, &in) {
		return
	}
	created, err := s.orders.checkout(in.OrderID, userID, in.SKU, in.Quantity, s.now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *storeServer) fulfill(w http.ResponseWriter, r *http.Request, userID string) {
	updated, err := s.orders.fulfill(r.PathValue("id"), userID, s.now())
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *storeServer) ordersForCustomer(w http.ResponseWriter, r *http.Request, userID string) {
	writeJSON(w, http.StatusOK, s.orders.list(userID))
}

func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
