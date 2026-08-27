package combos

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

// Credentials are HMAC API credentials used by account or builder Combo APIs.
type Credentials struct {
	Key        string
	Secret     string
	Passphrase string
}

// AccountAuth identifies the requester account for authenticated Combo calls.
type AccountAuth struct {
	Address     string
	Credentials Credentials
}

// BuilderAuth identifies the builder for Combo RFQ Gateway calls.
type BuilderAuth struct {
	Credentials Credentials
}

func buildAccountHeaders(account AccountAuth, method, fullPath, body string, timestamp int64) (http.Header, error) {
	address := strings.TrimSpace(account.Address)
	if address == "" {
		return nil, fmt.Errorf("account address is required")
	}
	headers, err := buildHMACHeaders(account.Credentials, method, fullPath, body, timestamp, false)
	if err != nil {
		return nil, err
	}
	headers.Set(auth.HeaderPolyAddress, address)
	return headers, nil
}

func buildBuilderHeaders(builder BuilderAuth, method, fullPath, body string, timestamp int64) (http.Header, error) {
	return buildHMACHeaders(builder.Credentials, method, fullPath, body, timestamp, true)
}

func buildHMACHeaders(creds Credentials, method, fullPath, body string, timestamp int64, builder bool) (http.Header, error) {
	if strings.TrimSpace(creds.Key) == "" || strings.TrimSpace(creds.Secret) == "" || strings.TrimSpace(creds.Passphrase) == "" {
		return nil, fmt.Errorf("api credentials are required")
	}
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	normalizedMethod := strings.ToUpper(method)
	message := strconv.FormatInt(timestamp, 10) + normalizedMethod + fullPath + body
	signature, err := auth.SignHMAC(creds.Secret, message)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	if builder {
		headers.Set(auth.HeaderPolyBuilderAPIKey, creds.Key)
		headers.Set(auth.HeaderPolyBuilderPassphrase, creds.Passphrase)
		headers.Set(auth.HeaderPolyBuilderTimestamp, strconv.FormatInt(timestamp, 10))
		headers.Set(auth.HeaderPolyBuilderSignature, signature)
		return headers, nil
	}
	headers.Set(auth.HeaderPolyAPIKey, creds.Key)
	headers.Set(auth.HeaderPolyPassphrase, creds.Passphrase)
	headers.Set(auth.HeaderPolyTimestamp, strconv.FormatInt(timestamp, 10))
	headers.Set(auth.HeaderPolySignature, signature)
	return headers, nil
}

func mergeHeaders(target map[string]string, source http.Header) {
	for key, values := range source {
		if len(values) == 0 {
			continue
		}
		target[key] = values[0]
	}
}
