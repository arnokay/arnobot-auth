package service

import (
	"time"

	"arnobot-shared/service"
)

type Services struct {
	TwitchService      *TwitchApiService
	ProviderService    *AuthProviderService
	UserService        *UserService
	SessionService     *SessionService
	TransactionService service.ITransactionService
}

type ProviderToken struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scopes       []string
	Expiry       time.Time
}
