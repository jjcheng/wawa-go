package feature_account_setting

// type Update struct {
// 	Get
// 	Value string `json:"value" val:"required" description:"value of the setting" example:"You are...."`
// }

// func (update *Update) Validate() []exception.InputException {
// 	update.Value = strings.TrimSpace(update.Value)
// 	errors := update.Get.Validate()
// 	if len(update.Value) < 100 {
// 		errors = append(errors, exception.NewInputException("value", "length of value must be >= 100"))
// 	}
// 	return errors
// }

// func (update Update) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_ai.Setting] {
// 	if cfg.Default().Site.Environment != types.EnvironmentDevelop && user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[*dto_ai.Setting](http.StatusUnauthorized, "you are not admin user")
// 	}
// 	if errors := update.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_ai.Setting](errors)
// 	}
// 	existing, ex := dependencies.UnitOfWork.AISettingRepository().GetByAppId(ctx, update.Id, user.App.Id)
// 	if ex != nil {
// 		if ex.StatusCode == http.StatusNotFound {
// 			return dto.NewFailedResponse[*dto_ai.Setting](ex.StatusCode, "setting not found")
// 		}
// 		return dto.NewFailedResponse[*dto_ai.Setting](ex.StatusCode, ex.Message)
// 	}
// 	// only can update value
// 	existing.Value = update.Value
// 	err := dependencies.UnitOfWork.AISettingRepository().Update(ctx, existing)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, user.Id, update)
// 		return dto.NewFailedResponse[*dto_ai.Setting](http.StatusInternalServerError, "error updating setting")
// 	}
// 	d := dto_ai.NewSetting(*existing)
// 	return dto.NewSuccessResponse(&d)
// }

// func (Update) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Update setting", "Update a setting.", types.HttpRequestTypeUriJSON, "PUT", "/ai/v1/settings/:id", true, false, types.APITagAI, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("setting not found", http.StatusNotFound)),
// 		feature.NewAPIError(*exception.NewCustomException("name already exists", http.StatusBadRequest)),
// 	})
// }
