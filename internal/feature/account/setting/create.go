package feature_account_setting

// type Create struct {
// 	Name  string `json:"name" val:"required" description:"name of the setting" example:"REWRITE_SYSTEM_MESSAGE"`
// 	Value string `json:"value" val:"required" description:"value of the setting" example:"You are...."`
// }

// func (create *Create) Validate() []exception.InputException {
// 	create.Name = strings.TrimSpace(create.Name)
// 	create.Value = strings.TrimSpace(create.Value)
// 	errors := []exception.InputException{}
// 	if create.Name == "" {
// 		errors = append(errors, exception.NewInputException("name", "missing name"))
// 	}
// 	if create.Value == "" {
// 		errors = append(errors, exception.NewInputException("value", "missing value"))
// 	}
// 	return errors
// }

// func (create Create) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[*dto_account.Setting] {
// 	if cfg.Default().Site.Environment != types.EnvironmentDevelop && user.Type != types.UserTypeAdmin {
// 		return dto.NewFailedResponse[*dto_account.Setting](http.StatusUnauthorized, "you are not admin user")
// 	}
// 	if errors := create.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[*dto_account.Setting](errors)
// 	}
// 	//check label already exist
// 	existing, _ := dependencies.UnitOfWork.AccountSettingRepository().GetByNameAndOrganizationId(ctx, create.Name, user.App.Id)
// 	if existing != nil {
// 		return dto.NewFailedResponse[*dto_ai.Setting](http.StatusBadRequest, "name exists")
// 	}
// 	new := dto_account.Setting{}
// 	new.Name = create.Name
// 	new.Value = create.Value
// 	//new.AppId = user.App.Id
// 	err := dependencies.UnitOfWork.AccountSettingRepository().Insert(ctx, &new)
// 	if err != nil {
// 		dependencies.Logger.ErrorFunction(err, user.Id, create)
// 		return dto.NewFailedResponse[*dto_ai.Setting](http.StatusInternalServerError, "error inserting setting")
// 	}
// 	d := dto_ai.NewSetting(new)
// 	return dto.NewSuccessResponse(&d)
// }

// func (Create) APISettings() feature.APISettings {
// 	return feature.NewAPISettings("Create setting", "Create a setting. Settings are mostly used by LLM actions.", types.HttpRequestTypeJSON, "POST", "/ai/v1/settings", true, false, types.APITagAI, []feature.APIError{
// 		feature.NewAPIError(*exception.NewCustomException("name exists", http.StatusBadRequest)),
// 	})
// }
