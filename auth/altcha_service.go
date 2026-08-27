package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go"
	"github.com/redis/go-redis/v9"
)

const (
	altchaChallengeLifetime = 2 * time.Minute
	altchaReplayLifetime    = 5 * time.Minute
	altchaReplayKeyPrefix   = "auth:altcha:used:"
)

// AltchaService creates and verifies locally hosted ALTCHA proof-of-work challenges.
// A verified challenge is atomically consumed in Redis so that it cannot be replayed.
type AltchaService struct {
	redisClient *redis.Client
	hmacKey     string
}

func NewAltchaService(redisClient *redis.Client, encryptService *EncryptService) *AltchaService {
	return &AltchaService{
		redisClient: redisClient,
		hmacKey:     base64.RawURLEncoding.EncodeToString(encryptService.DeriveHMACKey("auth/altcha/v1")),
	}
}

func (t *AltchaService) Alloc() (*altcha.Challenge, error) {
	expires := time.Now().Add(altchaChallengeLifetime)
	challenge, err := altcha.CreateChallenge(altcha.ChallengeOptions{
		Algorithm: altcha.SHA256,
		HMACKey:   t.hmacKey,
		MaxNumber: 100000,
		Expires:   &expires,
	})
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (t *AltchaService) Validate(ctx context.Context, payload string) (bool, error) {
	ok, err := altcha.VerifySolutionSafe(payload, t.hmacKey, true)
	if err != nil || !ok {
		return false, nil
	}

	challengeSignature, ok := altchaPayloadSignature(payload)
	if !ok {
		return false, nil
	}

	used, err := t.redisClient.SetNX(ctx, altchaReplayKeyPrefix+challengeSignature, "1", altchaReplayLifetime).Result()
	if err != nil {
		return false, err
	}
	return used, nil
}

func altchaPayloadSignature(payload string) (string, bool) {
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return "", false
		}
	}

	var value struct {
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(decoded, &value); err != nil || value.Signature == "" {
		return "", false
	}

	// Hash before using the signature as a cache-key component to keep the key format bounded.
	sum := sha256.Sum256([]byte(value.Signature))
	return hex.EncodeToString(sum[:]), true
}
