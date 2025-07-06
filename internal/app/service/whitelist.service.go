package service

import (
	"context"
	

	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/data"
	"github.com/arnokay/arnobot-shared/storage"
)

type WhitelistService struct {
	store  storage.Storager
	logger applog.Logger
}

func NewWhitelistService(store storage.Storager) *WhitelistService {
	logger := applog.NewServiceLogger("whitelist-service")

	return &WhitelistService{
		store:  store,
		logger: logger,
	}
}

func (s *WhitelistService) GetOne(ctx context.Context, d data.WhitelistGetOne) (data.Whitelist, error) {
	fromDB, err := s.store.Query(ctx).WhitelistGetOne(ctx, d.ToDB())
	if err != nil {
		s.logger.DebugContext(
			ctx,
			"cannot find in whitelist",
			"platform", d.Platform,
			"uid", d.UserID,
			"puid", d.PlatformUserID,
			"pun", d.PlatformUserName,
			"pul", d.PlatformUserLogin,
		)
		return data.Whitelist{}, s.store.HandleErr(ctx, err)
	}

	whitelist := data.NewWhitelistFromDB(fromDB)

	return whitelist, nil
}

func (s *WhitelistService) UpdateByID(ctx context.Context, id int32, d data.WhitelistUpdate) (data.Whitelist, error) {
	fromDB, err := s.store.Query(ctx).WhitelistUpdate(ctx, d.ToDB(id))
	if err != nil {
		return data.Whitelist{}, s.store.HandleErr(ctx, err)
	}
	return data.NewWhitelistFromDB(fromDB), nil
}
