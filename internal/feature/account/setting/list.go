package feature_account_setting

// type List struct {
// }

// func (list *List) Validate() []exception.InputException {
// 	return []exception.InputException{}
// }

// func (list List) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[[]dto_ai.Setting] {
// 	if cfg.Default().Site.Environment != types.EnvironmentDevelop && user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[[]dto_ai.Setting](http.StatusUnauthorized, "you are not admin user")
// 	}
// 	if errors := list.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[[]dto_ai.Setting](errors)
// 	}
// 	results, err := dependencies.UnitOfWork.AISettingRepository().ListByAppId(ctx, user.App.Id)
// 	if err != nil {
// 		return dto.NewFailedResponse[[]dto_ai.Setting](err.StatusCode, err.Message)
// 	}
// 	items := []dto_ai.Setting{}
// 	for _, item := range results {
// 		setting := dto_ai.NewSetting(item)
// 		items = append(items, setting)
// 	}
// 	return dto.NewSuccessResponse(items)
// }

// func (List) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("List settings", "List all settings.", types.HttpRequestTypeQuery, "GET", "/ai/v1/settings", true, false, types.APITagAI, nil)
// }
