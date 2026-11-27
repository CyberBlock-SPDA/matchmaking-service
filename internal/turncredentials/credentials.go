package turncredentials

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type ICECredentials struct {
	ICEServers []ICEServer `json:"iceServers"`
}

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username"`
	Credential string   `json:"credential"`
}

type Service struct {
	urls   []string
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func New(urls []string, secret string, ttl time.Duration) (*Service, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("at least one TURN URL is required")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("TURN shared secret must contain at least 32 bytes")
	}
	if ttl <= 0 || ttl > time.Hour {
		return nil, fmt.Errorf("TURN credential TTL must be between 0 and 1h")
	}
	for _, url := range urls {
		if !strings.HasPrefix(url, "turn:") && !strings.HasPrefix(url, "turns:") {
			return nil, fmt.Errorf("invalid TURN URL scheme")
		}
	}

	return &Service{
		urls:   append([]string(nil), urls...),
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}, nil
}

func (s *Service) Credentials() (ICECredentials, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return ICECredentials{}, fmt.Errorf("generate TURN username: %w", err)
	}

	username := fmt.Sprintf(
		"%d:%s",
		s.now().Add(s.ttl).Unix(),
		hex.EncodeToString(random),
	)
	mac := hmac.New(sha1.New, s.secret)
	if _, err := mac.Write([]byte(username)); err != nil {
		return ICECredentials{}, fmt.Errorf("sign TURN username: %w", err)
	}

	return ICECredentials{
		ICEServers: []ICEServer{{
			URLs:       append([]string(nil), s.urls...),
			Username:   username,
			Credential: base64.StdEncoding.EncodeToString(mac.Sum(nil)),
		}},
	}, nil
}
