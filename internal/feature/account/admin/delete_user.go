package feature_account_admin

// type Delete struct {
// 	Identifier string `uri:"identifier" val:"required" description:"identifier of the user to delete" example:"bf4a7bf2-5433-4401-afb6-300e10e8f27e"`
// }

// func (delete *Delete) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if delete.Identifier == "" {
// 		errors = append(errors, exception.NewInputException("id", "invalid id"))
// 	}
// 	return errors
// }

// func (delete Delete) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[any] {
// 	if user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not admin")
// 	}
// 	if errors := delete.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[any](errors)
// 	}
// 	existingUser, ex := dependencies.UnitOfWork.AIUserRepository().GetByIdentifierAndAppId(ctx, user.App.Id, delete.Identifier)
// 	if ex != nil {
// 		if ex.StatusCode == http.StatusNotFound {
// 			return dto.NewFailedResponse[any](ex.StatusCode, "user not found")
// 		}
// 		return dto.NewFailedResponse[any](ex.StatusCode, "error getting user")
// 	}
// 	if existingUser.Type == types.UserTypeMaster {
// 		return dto.NewFailedResponse[any](http.StatusBadRequest, "master user cannot be deleted")
// 	}
// 	err := dependencies.UnitOfWork.AIUserRepository().DeleteById(ctx, existingUser.Id)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, user.Id, delete)
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, "error deleting user")
// 	}
// 	return dto.NewEmptyResponse(true, http.StatusOK)
// }

// func (Delete) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Delete user", "Delete a user by its identifier. Only admin can delete users.", types.HttpRequestTypeUri, "DELETE", "/users/v1/:identifier", true, false, types.APITagAccount, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
// 	})
// }
