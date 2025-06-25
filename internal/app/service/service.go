// application services
package service

import (
	"github.com/arnokay/arnobot-shared/service"
)

type Services struct {
  PlatformAPIService *PlatformAPIService
	ProviderService    *AuthProviderService
	UserService        *UserService
	SessionService     *SessionService
	TransactionService service.ITransactionService
	OAuthService       OAuthProvider
	WhitelistService   *WhitelistService
}
