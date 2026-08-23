package wechat

import "context"

// AccessToken exposes the phone client's shared stable-token cache to other
// official Mini Program APIs without creating a second credential client.
func (client *PhoneNumberClient) AccessToken(ctx context.Context) (string, error) {
	if client == nil || ctx == nil {
		return "", ErrUnavailable
	}
	token, err := client.stableToken(ctx)
	if err != nil || token == "" {
		return "", ErrUnavailable
	}
	return token, nil
}
