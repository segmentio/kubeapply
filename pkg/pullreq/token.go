package pullreq

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
)

// GenerateJWT generates a signed JWT for an app. See
// https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/
// for more details.
func GenerateJWT(pemStr string, appID string) (string, error) {
	now := time.Now()

	block, _ := pem.Decode([]byte(pemStr))
	if block.Bytes == nil {
		return "", fmt.Errorf("Could not parse pem string")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	jwtObj := jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		jwt.MapClaims{
			// Now in epoch seconds
			"iat": now.Unix(),
			// Max expiration time in epoch seconds; max is 10 minutes from now
			"exp": now.Add(10 * time.Minute).Unix(),
			// App ID
			"iss": appID,
		},
	)
	return jwtObj.SignedString(key)
}

// AccessToken wraps the results returned by the Github access_tokens API.
type AccessToken struct {
	Token       string            `json:"token"`
	ExpiresAt   time.Time         `json:"expires_at"`
	Permissions map[string]string `json:"permissions"`
}

// GenerateAccessToken uses a JWT to generate an access token in a specific app
// installation (e.g., organization). See
// https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/
// for more details.
func GenerateAccessToken(
	ctx context.Context,
	jwt string,
	installationID string,
) (*AccessToken, error) {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf(
			"https://api.github.com/app/installations/%s/access_tokens",
			installationID,
		),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	req.Header.Add("Accept", "application/vnd.github.machine-man-preview+json")
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"Non-2XX response (%d): %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	tokenObj := &AccessToken{}
	if err := json.Unmarshal(bodyBytes, tokenObj); err != nil {
		return nil, err
	}

	log.Infof(
		"Successfully generated github access token with expiration at %s",
		tokenObj.ExpiresAt,
	)

	return tokenObj, nil
}
