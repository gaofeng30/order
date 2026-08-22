package wechatpay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"time"
)

const responseMaxSkew = 5 * time.Minute

// SignatureHeaders are the four original Wechatpay response or callback headers.
type SignatureHeaders struct {
	Serial    string
	Signature string
	Timestamp string
	Nonce     string
}

func (client *Client) verify(body []byte, headers SignatureHeaders) error {
	publicKey := client.providerPublicKeys[headers.Serial]
	if publicKey == nil {
		return &Error{kind: ErrorUnknownSerial}
	}
	timestamp, err := strconv.ParseInt(headers.Timestamp, 10, 64)
	if err != nil {
		return &Error{kind: ErrorTimestamp}
	}
	delta := client.now().Sub(time.Unix(timestamp, 0))
	if delta < -responseMaxSkew || delta > responseMaxSkew {
		return &Error{kind: ErrorTimestamp}
	}
	if headers.Nonce == "" {
		return &Error{kind: ErrorProtocol}
	}
	signature, err := base64.StdEncoding.DecodeString(headers.Signature)
	if err != nil {
		return &Error{kind: ErrorSignature}
	}
	message := headers.Timestamp + "\n" + headers.Nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], signature); err != nil {
		return &Error{kind: ErrorSignature}
	}
	return nil
}

func signSHA256RSA(key *rsa.PrivateKey, message []byte) (string, error) {
	digest := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", &Error{kind: ErrorProtocol}
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func randomNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func safeHeaderToken(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character <= 0x20 || character >= 0x7f || character == '"' || character == '\\' {
			return false
		}
	}
	return true
}
