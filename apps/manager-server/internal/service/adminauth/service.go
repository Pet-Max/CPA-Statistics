package adminauth

import (
	"context"
	"errors"
	"time"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/config"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/security"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
)

type Service struct {
	cfg   config.Config
	store *store.Store
}

func New(cfg config.Config, store *store.Store) *Service {
	return &Service{cfg: cfg, store: store}
}

func (s *Service) VerifyHeader(ctx context.Context, authorizationHeader string) (bool, error) {
	credential, ok, err := s.store.LoadAdminCredential(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("admin credential is not initialized")
	}
	adminKey := security.ExtractBearerToken(authorizationHeader)
	if !security.VerifyAdminKey(credential, adminKey) {
		return false, nil
	}
	if security.AdminCredentialNeedsUpgrade(credential) {
		upgraded, err := security.NewAdminCredential(adminKey, credential.Source)
		if err != nil {
			return false, err
		}
		if credential.CreatedAtMS > 0 {
			upgraded.CreatedAtMS = credential.CreatedAtMS
		}
		upgraded.RotatedAtMS = time.Now().UnixMilli()
		if err := s.store.SaveAdminCredential(ctx, upgraded); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Service) VerifyPanelHeader(ctx context.Context, authorizationHeader string) (bool, error) {
	return s.VerifyHeader(ctx, authorizationHeader)
}

func (s *Service) VerifySubmittedExternalConfigHeader(ctx context.Context, authorizationHeader string, cfg store.ManagerConfig) (bool, error) {
	return s.VerifyHeader(ctx, authorizationHeader)
}
