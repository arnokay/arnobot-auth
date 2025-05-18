package service

import (
	"time"
)

type Services struct {
	TwitchService   *TwitchApiService
	ProviderService *AuthProviderService
	UserService     *UserService
	SessionService  *SessionService
}

type ProviderToken struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
}
