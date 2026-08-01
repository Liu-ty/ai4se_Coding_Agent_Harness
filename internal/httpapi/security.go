package httpapi

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type cryptoRandomReader struct{}

func (cryptoRandomReader) Read(buffer []byte) (int, error) {
	return rand.Read(buffer)
}

func (r *Router) randomToken() (string, error) {
	var value [32]byte
	r.randomMu.Lock()
	_, err := io.ReadFull(r.random, value[:])
	r.randomMu.Unlock()
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

func (r *Router) requestID() string {
	token, err := r.randomToken()
	if err != nil {
		return "req-unavailable"
	}
	return "req-" + token[:16]
}

func (r *Router) validHost(host string) bool {
	return strings.EqualFold(strings.TrimSpace(host), r.host)
}

func (r *Router) validOrigin(rawOrigin string) bool {
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Scheme != "http" || origin.User != nil ||
		origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}
	return strings.EqualFold(origin.Host, r.host)
}

func (r *Router) validSession(request *http.Request) bool {
	cookie, err := request.Cookie(sessionCookieName)
	return err == nil && secureEqual(cookie.Value, r.sessionToken)
}

func (r *Router) handleBootstrap(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
) {
	if request.Method != http.MethodGet {
		r.writeMethodNotAllowed(writer, requestID)
		return
	}
	token := request.URL.Query().Get("bootstrap")
	r.bootstrapMu.Lock()
	valid := !r.bootstrapUsed && secureEqual(token, r.bootstrapToken)
	if valid {
		r.bootstrapUsed = true
	}
	r.bootstrapMu.Unlock()
	if !valid {
		r.writeError(writer, http.StatusUnauthorized, "INVALID_BOOTSTRAP", "bootstrap token is invalid or already used", requestID)
		return
	}
	http.SetCookie(writer, &http.Cookie{
		Name: sessionCookieName, Value: r.sessionToken, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: false,
	})
	http.Redirect(writer, request, "/", http.StatusSeeOther)
}
