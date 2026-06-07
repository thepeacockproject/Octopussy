// Package jwt implements the server's JSON Web Token handling. Its design is
// inspired by Peacock's oauthToken module, but follows the newer
// IOI auth stack the Bond game expects: RS256-signed "at+jwt" tokens carrying
// the full IOI claim set, with the signing certificate's SHA-1 thumbprint
// exposed as the kid / x5t header values.
package jwt

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func init() {
	// IOI tokens serialize a single-element "aud" as a plain string, not an
	// array; match that so our tokens are byte-shaped like the real ones.
	jwt.MarshalSingleStringAsArray = false
}

// Claims is the token payload. It embeds the registered JWT claims (iss, aud,
// sub, exp, nbf, iat, jti) and adds the IOI-specific fields observed on the
// game's real access tokens.
type Claims struct {
	ClientID   string   `json:"client_id,omitempty"`
	Roles      string   `json:"roles,omitempty"`
	UniqueName string   `json:"unique_name,omitempty"`
	AuthMethod string   `json:"auth:method,omitempty"`
	Platform   string   `json:"platform,omitempty"`
	Country    string   `json:"cntry,omitempty"`
	Locale     string   `json:"locale,omitempty"`
	Region     string   `json:"rgn,omitempty"`
	GameServer string   `json:"gs,omitempty"`
	BuildID    string   `json:"bid,omitempty"`
	PIS        string   `json:"pis,omitempty"`
	UserID     string   `json:"userid,omitempty"`
	Scope      []string `json:"scope,omitempty"`
	RegisteredClaims
}

// RegisteredClaims re-exports the standard registered-claims struct so callers
// can populate iss/sub/aud without importing golang-jwt directly.
type RegisteredClaims = jwt.RegisteredClaims

// Audience builds the "aud" claim value from one or more audiences.
func Audience(auds ...string) jwt.ClaimStrings {
	return jwt.ClaimStrings(auds)
}

// NewNumericDate re-exports golang-jwt's constructor for exp/nbf/iat values so
// callers can set time-based claims without importing golang-jwt directly.
func NewNumericDate(t time.Time) *jwt.NumericDate {
	return jwt.NewNumericDate(t)
}

// Config controls how the Manager mints and validates tokens.
type Config struct {
	// PrivateKeyPEM is an optional PEM-encoded RSA private key (PKCS#1 or
	// PKCS#8). When empty, an ephemeral 2048-bit key is generated at startup —
	// fine for dev, since the same in-memory key both signs and verifies.
	PrivateKeyPEM []byte
	// TTL is how long a token stays valid. Defaults to 2 hours (7200s), which
	// matches the game's observed expires_in.
	TTL time.Duration
	// Logger is used for diagnostics. Defaults to slog.Default.
	Logger *slog.Logger
}

// Manager signs and verifies tokens with a single RSA key pair.
type Manager struct {
	key    *rsa.PrivateKey
	kid    string
	x5t    string
	ttl    time.Duration
	logger *slog.Logger
}

// NewManager builds a Manager, generating a signing key when one is not
// supplied and deriving the kid / x5t thumbprints from a self-signed cert.
func NewManager(cfg Config) (*Manager, error) {
	key, err := loadOrGenerateKey(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, err
	}

	kid, x5t, err := thumbprint(key)
	if err != nil {
		return nil, err
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 2 * time.Hour
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Manager{key: key, kid: kid, x5t: x5t, ttl: ttl, logger: logger}, nil
}

// loadOrGenerateKey parses a PEM key if given, otherwise generates a fresh one.
func loadOrGenerateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	if len(pemBytes) == 0 {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate rsa key: %w", err)
		}
		return key, nil
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("jwt: invalid PEM private key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("jwt: private key is not RSA")
	}
	return key, nil
}

// thumbprint builds a self-signed certificate around the key and returns the
// SHA-1 thumbprint of its DER encoding as (kid hex-upper, x5t base64url). This
// reproduces how the IOI tokens derive their kid and x5t header values.
func thumbprint(key *rsa.PrivateKey) (kid, x5t string, err error) {
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "octopussy-signing"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", fmt.Errorf("create signing cert: %w", err)
	}
	sum := sha1.Sum(der)
	return strings.ToUpper(hex.EncodeToString(sum[:])), base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// Sign mints a signed token string for the given claims. Time-based claims and
// the JWT ID are filled in automatically when left unset; the issuer, audience,
// subject, and IOI fields are the caller's responsibility (they vary per grant
// type). The header carries the at+jwt type and the kid / x5t thumbprints.
func (m *Manager) Sign(claims Claims) (string, error) {
	now := time.Now()
	if claims.IssuedAt == nil {
		claims.IssuedAt = jwt.NewNumericDate(now)
	}
	if claims.NotBefore == nil {
		claims.NotBefore = jwt.NewNumericDate(now)
	}
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(m.ttl))
	}
	if claims.ID == "" {
		jti, err := randomJTI()
		if err != nil {
			return "", err
		}
		claims.ID = jti
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["typ"] = "at+jwt"
	token.Header["kid"] = m.kid
	token.Header["x5t"] = m.x5t

	signed, err := token.SignedString(m.key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// randomJTI returns a 16-byte random ID as upper-case hex, matching the jti
// format on the real tokens.
func randomJTI() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}
	return strings.ToUpper(hex.EncodeToString(buf)), nil
}

// Parse validates a token string and returns its claims. Validation enforces
// the RS256 algorithm and the standard time-based claims; we trust our own
// signature rather than re-checking issuer/audience.
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return &m.key.PublicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// TTL reports the configured token lifetime, useful for "expires_in" responses.
func (m *Manager) TTL() time.Duration { return m.ttl }

// Kid returns the signing key's thumbprint identifier, for a future JWKS endpoint.
func (m *Manager) Kid() string { return m.kid }

// PublicKey returns the verification key, for a future JWKS endpoint.
func (m *Manager) PublicKey() *rsa.PublicKey { return &m.key.PublicKey }

// contextKey is unexported to avoid collisions in request context values.
type contextKey struct{}

var claimsKey = contextKey{}

// ErrMissingToken is returned when no bearer token is present on a request.
var ErrMissingToken = errors.New("missing bearer token")

// extractBearer pulls the token out of an "Authorization: Bearer <token>"
// header, mirroring Peacock's extractToken middleware.
func extractBearer(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", ErrMissingToken
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", ErrMissingToken
	}
	return strings.TrimSpace(parts[1]), nil
}

// Middleware validates the bearer token on each request and stores the parsed
// claims in the request context. Requests without a valid token receive 401.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := extractBearer(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.Parse(raw)
		if err != nil {
			m.logger.Debug("rejected token", slog.String("error", err.Error()))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFrom retrieves the claims stored by Middleware, if present.
func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}
