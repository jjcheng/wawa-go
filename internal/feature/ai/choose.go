package feture_ai

// type Choose struct {
// 	Messages []dto_wa.Message
// 	Questions service.DashScopeDecideQuestion
// }

// func (choose *Choose) Validate() []exception.InputException {
// 	var errors []exception.InputException
// 	if len(choose.Messages) == 0 {
// 		errors = append(errors, exception.NewInputException("messages", "missing messages"))
// 	}
// 	if len(choose.Choices) == 0 {
// 		errors = append(errors, exception.NewInputException("choices", "missing choices"))
// 	}
// 	return errors
// }

// func (choose Choose) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[string] {
// 	if errors := choose.Validate(); len(errors) > 0 {
// 		return dto.NewInvalidInputResponse[string](errors)
// 	}
// 	var serviceMessages []service.DashScopeMessage = make([]service.DashScopeMessage, len(choose.Messages))
// 	for i := range choose.Messages {
// 		text := choose.Messages[i].Text()
// 		if text == "" {
// 			continue
// 		}
// 		serviceMessage := service.DashScopeMessage{
// 			Content: text,
// 		}
// 		if choose.Messages[i].Sending {
// 			serviceMessage.Role = service.DashScopeChatMessageRoleAssistant
// 		} else {
// 			serviceMessage.Role = service.DashScopeChatMessageRoleUser
// 		}
// 	}
// 	questions := map[string]service.DashScopeDecideQuestionContent{}
// 	for k, v := range questions {
// 		questions[k] = service.DashScopeDecideQuestionContent{
// 			Type: service.DashScopeDecideQuestionTypeChoice,

// 		}
// 	}

// 	dependencies.LLM.Decide(ctx, serviceMessages, choose.Choices)
// }
