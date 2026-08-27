package feature_wa

import (
	"context"
	"net/http"

	"github.com/jjcheng/wawa-go/internal/exception"
	"github.com/jjcheng/wawa-go/internal/service"
)

func ClientForWABA(ctx context.Context, dependencies *service.Dependencies, wabaID string) (*service.Whatsapp, *exception.Exception) {
	businessAccount, ex := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, wabaID)
	if ex != nil {
		return nil, ex
	}
	businessPortfolio, ex := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, businessAccount.MetaBusinessPortfolioId)
	if ex != nil {
		return nil, ex
	}
	if businessPortfolio.AccessToken == "" {
		return nil, exception.NewCustomException("business access token is not configured", http.StatusBadGateway)
	}
	return dependencies.Whatsapp.WithBusinessAccessToken(businessPortfolio.AccessToken), nil
}
