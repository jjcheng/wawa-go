package feature_account_user

// type UpdateStatus struct {
// 	Id     int              `json:"id" val:"required" description:"id of the user"`
// 	Status types.UserStatus `json:"status" val:"required" description:"new status of the user"`
// }

// func (updateStatus *UpdateStatus) Validate() []exception.InputException {
// 	errors := []exception.InputException{}
// 	if updateStatus.Id < 0 {
// 		errors = append(errors, exception.NewInputException("id", "missing id"))
// 	}
// 	if updateStatus.Status == "" {
// 		errors = append(errors, exception.NewInputException("status", "missing status"))
// 	} else if updateStatus.Status == types.UserStatusPending {
// 		errors = append(errors, exception.NewInputException("status", "status must not be PENDING"))
// 	}
// 	return errors
// }

// func (updateStatus UpdateStatus) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[any] {
// 	if user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[any](http.StatusUnauthorized, "you are not admin")
// 	}
// 	if errors := updateStatus.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[any](errors)
// 	}
// 	existingUser, ex := dependencies.UnitOfWork.AccountUserRepository().GetByOrganizationId(ctx, int32(updateStatus.Id), user.Organization.Id)
// 	if ex != nil {
// 		if ex.StatusCode == http.StatusNotFound {
// 			return dto.NewFailedResponse[any](ex.StatusCode, "user not found")
// 		}
// 		return dto.NewFailedResponse[any](ex.StatusCode, "error getting user")
// 	}
// 	existingUser.Status = updateStatus.Status
// 	err := dependencies.UnitOfWork.AccountUserRepository().Update(ctx, existingUser)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, user.Id, updateStatus)
// 		return dto.NewFailedResponse[any](http.StatusInternalServerError, "error updating user status")
// 	}
// 	return dto.NewEmptyResponse(true, http.StatusOK)
// }

// func (UpdateStatus) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Update user status", "Enable or disable a user. Only admin can update user status.", types.HttpRequestTypeJSON, "PATCH", "/user/v1/status", true, false, types.APITagAccount, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("user not found", http.StatusNotFound)),
// 	})
// }
