package feature_account_setting

// type Delete struct {
// 	Get
// }

// func (delete *Delete) Validate() []exception.InputException {
// 	errors := delete.Get.Validate()
// 	return errors
// }

// func (delete Delete) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
// 	if cfg.Default().Site.Environment != types.EnvironmentDevelop && user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not admin user")
// 	}
// 	if errors := delete.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[any](errors)
// 	}
// 	existing, ex := dependencies.UnitOfWork.AISettingRepository().GetByAppId(ctx, delete.Id, user.App.Id)
// 	if ex != nil {
// 		if ex.StatusCode == http.StatusNotFound {
// 			return dto.NewFailedResponse[any](ex.StatusCode, "setting not found")
// 		}
// 		return dto.NewFailedResponse[any](ex.StatusCode, ex.Message)
// 	}
// 	err := dependencies.UnitOfWork.AISettingRepository().DeleteById(ctx, existing.Id)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, user.Id, delete)
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, "error deleting setting")
// 	}
// 	return dto.NewEmptyResponse(true, http.StatusOK)
// }

// func (Delete) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Delete setting", "Delete a setting.", types.HttpRequestTypeUri, "DELETE", "/ai/v1/settings/:id", true, false, types.APITagAI, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("setting not found", http.StatusNotFound)),
// 	})
// }
