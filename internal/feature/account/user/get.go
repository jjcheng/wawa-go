package feature_account_user

// // this is for internal use only, to retrieve respective user based on Whatsapp webhook where a WABA id will be included, no auth
// type Get struct {
// 	Identifier string `uri:"identifier" val:"required" description:"identifier of the user to delete" example:"bf4a7bf2-5433-4401-afb6-300e10e8f27e"`
// }

// func (get *Get) Validate() []exception.InputException {
// 	get.Identifier = strings.TrimSpace(get.Identifier)
// 	errors := []exception.InputException{}
// 	if get.Identifier == "" {
// 		errors = append(errors, exception.NewInputException("identifier", "missing identifier"))
// 	}
// 	return errors
// }

// func (get Get) Handle(ctx context.Context, _, dependencies *service.Dependencies) dto.Response[*dto_ai.User] {
// 	if errors := get.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_ai.User](errors)
// 	}
// 	existingUser, ex := dependencies.UnitOfWork.AIUserRepository().GetByIdentifier(ctx, get.Identifier)
// 	if ex != nil {
// 		if ex.StatusCode == http.StatusNotFound {
// 			return dto.NewFailedResponse[*dto_ai.User](ex.StatusCode, "user not found")
// 		}
// 		return dto.NewFailedResponse[*dto_ai.User](ex.StatusCode, "error getting user")
// 	}
// 	appDAO, err, notFound := dependencies.UnitOfWork.AIAppRepository().GetById(ctx, existingUser.AppId)
// 	if notFound {
// 		return dto.NewFailedResponse[*dto_ai.User](http.StatusNotFound, "app not found")
// 	}
// 	if err != nil {
// 		return dto.NewFailedResponse[*dto_ai.User](http.StatusInternalServerError, "error getting app")
// 	}
// 	d := dto_ai.NewUser(*existingUser, appDAO)
// 	d.APIKey = ""
// 	return dto.NewSuccessResponse(&d)
// }
