package feature_wa

// func ClientForWABA(ctx context.Context, dependencies *service.Dependencies, wabaID string) (*service.Whatsapp, *exception.Exception) {
// 	businessAccount, ex := dependencies.UnitOfWork.WABusinessAccountRepository().GetByMetaWABAId(ctx, wabaID)
// 	if ex != nil {
// 		return nil, ex
// 	}
// 	return ClientForBusinessPortfolio(ctx, dependencies, businessAccount.MetaBusinessPortfolioId)
// }

// func ClientForBusinessPortfolio(ctx context.Context, dependencies *service.Dependencies, portfolioID string) (*service.Whatsapp, *exception.Exception) {
// 	portfolioID = strings.TrimSpace(portfolioID)
// 	if portfolioID == "" {
// 		return nil, exception.NewCustomException("business portfolio ID is required", http.StatusBadRequest)
// 	}
// 	businessPortfolio, ex := dependencies.UnitOfWork.WABusinessPortfolioRepository().GetByMetaBusinessPortfolioId(ctx, portfolioID)
// 	if ex != nil {
// 		return nil, ex
// 	}
// 	if strings.TrimSpace(businessPortfolio.AccessToken) == "" {
// 		return nil, exception.NewCustomException("business access token is not configured", http.StatusBadGateway)
// 	}
// 	if businessPortfolio.AccessTokenExpiresIn > 0 &&
// 		!businessPortfolio.EntryDate.IsZero() &&
// 		time.Now().After(businessPortfolio.EntryDate.Add(time.Duration(businessPortfolio.AccessTokenExpiresIn)*time.Second)) {
// 		return nil, exception.NewCustomException("business access token expired; reconnect the WhatsApp account", http.StatusUnauthorized)
// 	}
// 	return dependencies.Whatsapp.WithBusinessAccessToken(businessPortfolio.AccessToken), nil
// }

// func ClientForPhoneNumber(ctx context.Context, dependencies *service.Dependencies, phoneNumberID int32) (*service.Whatsapp, *exception.Exception) {
// 	phoneNumber, ex := dependencies.UnitOfWork.WAPhoneNumberRepository().Get(ctx, phoneNumberID)
// 	if ex != nil {
// 		return nil, ex
// 	}
// 	return ClientForBusinessPortfolio(ctx, dependencies, phoneNumber.MetaBusinessPortfolioId)
// }
