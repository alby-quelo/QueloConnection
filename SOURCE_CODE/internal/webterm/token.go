package webterm

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ValidateToken checks an HMAC token: base64url(machine|exp|sig).
func ValidateToken(token, machine, adminSecret string) error {
	if adminSecret == "" {
		return fmt.Errorf("admin token not configured")
	}
	if token == "" {
		return fmt.Errorf("missing token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("invalid token encoding")
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return fmt.Errorf("invalid token format")
	}
	tMachine := parts[0]
	expStr := parts[1]
	sig := parts[2]
	if tMachine != machine {
		return fmt.Errorf("token machine mismatch")
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid token expiry")
	}
	if time.Now().Unix() > exp {
		return fmt.Errorf("token expired")
	}
	data := tMachine + "|" + expStr
	mac := hmac.New(sha256.New, []byte(adminSecret))
	_, _ = mac.Write([]byte(data))
	expected := fmt.Sprintf("%x", mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return fmt.Errorf("invalid token signature")
	}
	return nil
}
