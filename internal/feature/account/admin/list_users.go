package feature_account_admin

// type List struct {
// }

// func (list *List) Validate() []exception.InputException {
// 	return nil
// }

// func (list List) Handle(ctx context.Context, user *dto_ai.User, dependencies *service.Dependencies) dto.Response[[]dto_ai.User] {
// 	if user.Type != types.UserTypeMaster && user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[[]dto_ai.User](http.StatusUnauthorized, "you are not admin")
// 	}
// 	if errors := list.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[[]dto_ai.User](errors)
// 	}
// 	var items []dto_ai.User
// 	userDAOs, ex := dependencies.UnitOfWork.AIUserRepository().ListByAppId(ctx, user.App.Id)
// 	if ex != nil {
// 		return dto.NewFailedResponse[[]dto_ai.User](ex.StatusCode, ex.Message)
// 	}
// 	for i := range userDAOs {
// 		items = append(items, dto_ai.NewUser(userDAOs[i], nil))
// 	}
// 	return dto.NewSuccessResponse(items)
// }

// func (List) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("List users", "List all users with their associated apps. Only admin can list users.", types.HttpRequestTypeQuery, "GET", "/users/v1", true, false, types.APITagAccount, nil)
// }
