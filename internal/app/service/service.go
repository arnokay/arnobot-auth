// application services
package service

import (
	"github.com/arnokay/arnobot-shared/service"
)

type Services struct {
	TwitchAPIService   *TwitchAPIService
	ProviderService    *AuthProviderService
	UserService        *UserService
	SessionService     *SessionService
	TransactionService service.ITransactionService
	TwitchOAuthService OAuthProvider
	WhitelistService   *WhitelistService
}
