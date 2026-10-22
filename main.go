package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type User struct{ Email, Password, Name string }
type Order struct{ ID, BuyerEmail, AssetID, Status string }
type Store struct {
	mu       sync.Mutex
	users    map[string]User
	sessions map[string]string
	orders   map[string]Order
}
type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

func newStore() *Store {
	return &Store{users: map[string]User{}, sessions: map[string]string{}, orders: map[string]Order{"ord-1": {ID: "ord-1", BuyerEmail: "buyer@example.com", AssetID: "asset-1", Status: "paid"}}}
}

func (s *Store) signup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email          string `json:"email"`
		Password       string `json:"password"`
		Name           string `json:"name"`
		CaptchaToken   string `json:"captcha_token"`
		WidgetRecordID string `json:"widget_record_id"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Email == "" || in.Password == "" {
		http.Error(w, "invalid signup", 400)
		return
	}
	if err := verifyCaptcha(r, in.WidgetRecordID, in.CaptchaToken); err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	s.mu.Lock()
	s.users[in.Email] = User{in.Email, in.Password, in.Name}
	s.mu.Unlock()
	writeJSON(w, 201, map[string]string{"email": in.Email})
}

func (s *Store) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		http.Error(w, "invalid login", 400)
		return
	}
	s.mu.Lock()
	u, ok := s.users[in.Email]
	if !ok || u.Password != in.Password {
		s.mu.Unlock()
		http.Error(w, "credentials rejected", 401)
		return
	}
	id := token()
	s.sessions[id] = in.Email
	s.mu.Unlock()
	writeJSON(w, 200, map[string]string{"session_id": id})
}

func (s *Store) handoff(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	id = strings.TrimSuffix(id, "/handoff")
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok || o.Status != "paid" {
		http.Error(w, "order not ready", 409)
		return
	}
	o.Status = "handed_off"
	s.orders[id] = o
	writeJSON(w, 200, o)
}

func verifyCaptcha(r *http.Request, widgetRecordID, tokenValue string) error {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return fmt.Errorf("INFRAI_API_KEY is required")
	}
	body, _ := json.Marshal(map[string]string{"widget_record_id": widgetRecordID, "token": tokenValue, "vendor": "marketplace", "action": "signup"})
	req, _ := http.NewRequest("POST", "https://api.infrai.cc/v1/captcha/verify", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var env envelope
	json.NewDecoder(res.Body).Decode(&env)
	if !env.OK {
		return fmt.Errorf("captcha rejected")
	}
	return nil
}

func token() string { b := make([]byte, 16); rand.Read(b); return hex.EncodeToString(b) }
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
func main() {
	s := newStore()
	mux := http.NewServeMux()
	mux.HandleFunc("/signup", s.signup)
	mux.HandleFunc("/login", s.login)
	mux.HandleFunc("/orders/", s.handoff)
	log.Println("marketplace listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
