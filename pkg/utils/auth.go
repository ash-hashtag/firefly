package utils

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type GooglePublicKey struct {
	Kid string `json:"kid"`
	E   string `json:"e"`
	N   string `json:"n"`
}

type GooglePublicKeys struct {
	Keys []GooglePublicKey `json:"keys"`
}

type VerifiedToken struct {
	Username    string
	Expires     time.Time
	Permissions int
}

func Base64URLDecode(input string) ([]byte, error) {
	return base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(input)
}

func JWKToRSAPublicKey(jwk GooglePublicKey) (*rsa.PublicKey, error) {
	// Decode modulus (n)
	nBytes, err := Base64URLDecode(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %v from %s", err, jwk.N)
	}

	// Decode exponent (e)
	eBytes, err := Base64URLDecode(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %v", err)
	}

	// Convert exponent from bytes to int
	e := int(binary.BigEndian.Uint32(append(make([]byte, 4-len(eBytes)), eBytes...)))

	// Construct RSA public key
	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}
	return pubKey, nil
}

func (keys *GooglePublicKeysHandler) JwtKeyFunc() jwt.Keyfunc {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		kid := token.Header["kid"]

		if reflect.TypeOf(kid).Kind() != reflect.String {
			return nil, fmt.Errorf("missing 'kid' in jwt header")
		}

		keys.mu.RLock()
		defer keys.mu.RUnlock()

		return keys.GetKeyWithKid(fmt.Sprintf("%s", kid)), nil

	}
	return keyFunc
}

func (keys *GooglePublicKeys) Verify(token string) (VerifiedToken, error) {
	verified_token := VerifiedToken{}
	segments := strings.SplitN(token, ".", 3)

	if len(segments) != 3 {
		return verified_token, fmt.Errorf("Invalid JWT Token")
	}

	return verified_token, nil
}

type GooglePublicKeysHandler struct {
	Keys             []GoogleRsaPublicKey
	expiresAt        time.Time
	mu               *sync.RWMutex
	expectedAudience string
}

type GoogleRsaPublicKey struct {
	Kid string
	key *rsa.PublicKey
}

func NewGooglePublicKeysHandler(expectedAudience string) GooglePublicKeysHandler {
	keys := GooglePublicKeysHandler{
		Keys:             make([]GoogleRsaPublicKey, 0, 4),
		expiresAt:        time.Now(),
		mu:               &sync.RWMutex{},
		expectedAudience: expectedAudience,
	}

	keys.GetKeys()

	return keys
}

const GOOGLE_PUBLIC_KEYS_URL = "https://www.googleapis.com/robot/v1/metadata/jwk/securetoken@system.gserviceaccount.com"

func ParseCacheControl(cacheControl string) int {

	for _, segment := range strings.Split(cacheControl, ",") {
		value, found := strings.CutPrefix(strings.Trim(segment, " "), "max-age=")
		if !found {
			continue
		}
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return -1
		}

		return seconds
	}

	return -1
}

type TokenPayload struct {
	Username    string `json:"uname"`
	Permissions int64  `json:"perms"`
	Expires     int64  `json:"exp"`
	IssuedAt    int64  `json:"iss"`
	Audience    string `json:"aud"`
}

type JWTHeader struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
}

func (self *GooglePublicKeysHandler) GetKeyWithKid(kid string) *rsa.PublicKey {
	self.mu.RLock()
	defer self.mu.RUnlock()

	for _, key := range self.Keys {
		if kid == key.Kid {
			return key.key
		}
	}

	return nil

}

func (self *GooglePublicKeysHandler) VerifyToken(token string) *VerifiedToken {

	if self.expiresAt.UnixMilli() < time.Now().UnixMilli() {
		self.GetKeys()
	}

	segments := strings.SplitN(token, ".", 3)
	if len(segments) != 3 {
		return nil
	}
	header, err := Base64URLDecode(segments[0])
	if err != nil {
		return nil
	}

	tokenHeader := JWTHeader{}
	if err := json.Unmarshal(header, &tokenHeader); err != nil {
		log.Printf("Couldn't parse JWT Header %s", header)
	}

	key := self.GetKeyWithKid(tokenHeader.Kid)
	if key == nil {
		return nil
	}

	signingPayload := segments[0] + "." + segments[1]

	signature, err := Base64URLDecode(segments[2])
	if err != nil {
		return nil
	}

	hash := sha256.Sum256([]byte(signingPayload))

	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], signature); err != nil {
		log.Printf("Failed to verify token %s", err)
		return nil
	}

	payload, err := Base64URLDecode(segments[1])
	if err != nil {
		return nil
	}

	tokenPayload := TokenPayload{}

	if err := json.Unmarshal(payload, &tokenPayload); err != nil {
		log.Printf("Failed to parse payload of token %s", payload)
		return nil
	}

	if tokenPayload.Expires < time.Now().Unix() {
		log.Printf("Failed to parse payload of token expired")
		return nil
	}

	if tokenPayload.Audience != self.expectedAudience {
		log.Printf("Invalid Audience")
		return nil
	}

	return nil
}

func (self *GooglePublicKeysHandler) GetKeys() error {
	self.mu.Lock()
	defer self.mu.Unlock()

	resp, err := http.Get(GOOGLE_PUBLIC_KEYS_URL)
	if err != nil {
		return err
	}

	expiresHeader := resp.Header.Get("expires")
	expiresAt, err := time.Parse(time.RFC1123, expiresHeader)
	if err == nil {
		self.expiresAt = expiresAt
	} else {
		cacheControl := resp.Header.Get("cache-control")
		if len(cacheControl) == 0 {
			return fmt.Errorf("Missing Cache Control Header")
		}

		seconds := ParseCacheControl(cacheControl)
		if seconds >= 0 {
			dateHeader := resp.Header.Get("date")
			date, err := time.Parse(time.RFC1123, dateHeader)
			if err == nil {
				self.expiresAt = date.Add(time.Duration(seconds) * time.Second)
			}
		}

	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := self.FillFromJson(body); err != nil {
		return err
	}

	return nil
}

func (self *GooglePublicKeysHandler) FillFromJson(jsonString []byte) error {
	newKeys := GooglePublicKeys{}

	if err := json.Unmarshal(jsonString, &newKeys); err != nil {
		return err
	}

	clear(self.Keys)

	for _, key := range newKeys.Keys {
		publicKey, err := JWKToRSAPublicKey(key)
		if err != nil {
			return err
		}

		self.Keys = append(self.Keys, GoogleRsaPublicKey{Kid: key.Kid, key: publicKey})

	}

	return nil
}
