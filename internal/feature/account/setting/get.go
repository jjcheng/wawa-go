package feature_account_setting

// type Get struct {
// 	Id int32 `uri:"id" val:"required" description:"id of the setting" example:"1"`
// }

// func (get *Get) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if get.Id <= 0 {
// 		errors = append(errors, exception.NewInputException("id", "missing id"))
// 	}
// 	return errors
// }

// func (get Get) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai.Setting] {
// 	if cfg.Default().Site.Environment != types.EnvironmentDevelop && user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[*dto_ai.Setting](http.StatusUnauthorized, "you are not admin user")
// 	}
// 	if errors := get.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_ai.Setting](errors)
// 	}
// 	result, ex := dependencies.UnitOfWork.AISettingRepository().GetByAppId(ctx, get.Id, user.App.Id)
// 	if ex != nil {
// 		if ex.StatusCode == http.StatusNotFound {
// 			return dto.NewFailedResponse[*dto_ai.Setting](ex.StatusCode, "setting not found")
// 		}
// 		return dto.NewFailedResponse[*dto_ai.Setting](ex.StatusCode, ex.Message)
// 	}
// 	d := dto_ai.NewSetting(*result)
// 	return dto.NewSuccessResponse(&d)
// }

// func (Get) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Get setting", "Get a setting.", types.HttpRequestTypeUri, "GET", "/ai/v1/settings/:id", true, false, types.APITagAI, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("setting not found", http.StatusNotFound)),
// 	})
// }
