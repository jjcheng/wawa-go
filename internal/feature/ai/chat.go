package feture_ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	dto_ai_agent "github.com/jjcheng/wawa-go/internal/dto/ai_agent"
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	"github.com/jjcheng/wawa-go/internal/exception"
	feature_ai_agent_skill "github.com/jjcheng/wawa-go/internal/feature/ai_agent/skill"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"
)

type Chat struct {
	Profile       *dto_ai_agent.Profile
	Messages      []dto_wa.Message
	MinSkillScore float32
}

func (chat *Chat) Validate() []exception.InputException {
	var errors []exception.InputException
	if chat.Profile == nil {
		errors = append(errors, exception.NewInputException("agent_profile", "missing agent profile"))
	}
	if len(chat.Messages) == 0 {
		errors = append(errors, exception.NewInputException("messages", "missing messages"))
	}
	if chat.MinSkillScore <= 0 {
		chat.MinSkillScore = 0.2
	}
	return errors
}

func (chat Chat) Handle(ctx context.Context, user *dto_account.User, dependencies *service.Dependencies) dto.Response[string] {
	if errors := chat.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[string](errors)
	}
	// convert messages to service messages
	var serviceMessages []service.DashScopeMessage = make([]service.DashScopeMessage, len(chat.Messages))
	for i := range chat.Messages {
		serviceMessages[i] = service.DashScopeMessage{
			Content: chat.Messages[i].Text(),
		}
		if chat.Messages[i].Sending {
			serviceMessages[i].Role = service.DashScopeChatMessageRoleAssistant
		} else {
			serviceMessages[i].Role = service.DashScopeChatMessageRoleUser
		}
	}
	// step 1: check if the message is spam, scam, nonsense or inapproriate
	// if so, end
	inValid, err := chat.checkIsInvalid(ctx, serviceMessages, dependencies)
	if err != nil {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if inValid {
		return dto.NewSuccessResponse("")
	}
	// step 2: check need to decompose tasks
	needToDecompose, err := chat.checkNeedToDecompose(ctx, serviceMessages, dependencies)
	if err != nil {
		return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	// step 3: if no need to decompose or rewrite, check skills and tool calling
	if !needToDecompose {
		message, err := chat.executeTask(ctx, serviceMessages, dependencies)
		if err != nil {
			return dto.NewFailedResponse[string](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		return dto.NewSuccessResponse(message)
	} else {
		// TODO: rewrite to decompose tasks
		// execute each task parallelly

	}
	return dto.NewSuccessResponse("Someone will assist you shortly.")
}

func (chat *Chat) checkIsInvalid(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) (bool, error) {
	// "department": {"type": "choice", "instructions": "应该由哪个团队处理？",
	//                 "criteria": {"billing": "支付、退款和账单问题", "technical": "产品故障和集成问题"}},
	// "escalate":   {"type": "noul",  "instructions": "是否需要立即通知值班人员？"},
	decideResponse, err := dependencies.LLM.Decide(ctx, messages, service.DashScopeDecideQuestion{
		"is_invalid": service.DashScopeDecideQuestionContent{
			Type:         service.DashScopeDecideQuestionTypeNoul,
			Instructions: "Is user's latest message a spam, scam, nonsense or inapproriate?",
			Criteria: service.DashScopeDecideQuestionCriteriaNoul{
				True:  "User's latest message is spam, scam, nonsense or inapproriate",
				False: "User's latest message is not spam, scam, nonsense or inapproriate",
			},
		},
	}, true)
	if err != nil {
		return false, err
	}
	// if it is scam, spam, nonsense, or inapproriate, end the conversation
	answer, ok := decideResponse.Answers["is_invalid"]
	if !ok || answer == nil {
		return false, fmt.Errorf("error getting answer from decide, is_valid is not found")
	}
	noul, err := helper.ConvertJSON[service.DashScopeDecideResponseAnswerNoul](answer)
	if err != nil {
		return false, fmt.Errorf("error converting answer from decide, is_valid: %w", err)
	}
	if noul.Noul > 0.5 {
		return true, nil
	}
	return false, nil
}

func (chat *Chat) detectSkills(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) ([]dto_ai_agent.Skill, error) {
	listSkills := feature_ai_agent_skill.List{
		ProfileId:   chat.Profile.Id,
		FromWebhook: true,
	}
	// this will return builtin skills
	listSkillsResponse := listSkills.Handle(ctx, nil, dependencies)
	if !listSkillsResponse.Success {
		return nil, listSkillsResponse.Error
	}
	if len(listSkillsResponse.Data) == 0 {
		return nil, nil
	}
	// if only 1 skill, return it
	if len(listSkillsResponse.Data) == 1 {
		return listSkillsResponse.Data, nil
	}
	// build criteria
	criteria := helper.Map(listSkillsResponse.Data, func(s dto_ai_agent.Skill) string {
		return s.Title
	})
	criteria = append(criteria, "None of above")
	skillScores, err := dependencies.LLM.Decide(ctx, messages, service.DashScopeDecideQuestion{
		"detect_skills": service.DashScopeDecideQuestionContent{
			Type:         service.DashScopeDecideQuestionTypeScore,
			Instructions: "Which of the criteria should be applied to user's latest message?",
			Criteria:     criteria,
		},
	}, true)
	if err != nil {
		return nil, fmt.Errorf("error detecting skill scores: %w", err)
	}
	if detectSkills, ok := skillScores.Answers["detect_skills"]; ok {
		scores, err := helper.ConvertJSON[service.DashScopeDecideResponseAnswerScore](detectSkills)
		if err != nil {
			return nil, fmt.Errorf("error parsing detect_skills: %w", err)
		}
		// rank scores.Probabilities by highest value on top
		type tuple struct {
			skillName string
			score     float32
		}
		var orderedSkills []tuple
		for k, v := range scores.Probabilities {
			if v < chat.MinSkillScore {
				continue
			}
			orderedSkills = append(orderedSkills, tuple{
				skillName: scores.Legend[k],
				score:     v,
			})
		}
		helper.Sort(orderedSkills, func(a, b tuple) int {
			if a.score > b.score {
				return -1
			} else if a.score < b.score {
				return 1
			} else {
				return 0
			}
		})
		skills := helper.Map(orderedSkills, func(t tuple) dto_ai_agent.Skill {
			match := helper.First(listSkillsResponse.Data, func(s dto_ai_agent.Skill) bool {
				return s.Title == t.skillName
			})
			return *match
		})
		return skills, nil
	} else {
		return nil, fmt.Errorf("detect skills not found in detectSkill function")
	}
}

func (chat *Chat) checkNeedToDecompose(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) (bool, error) {
	decideResponse, err := dependencies.LLM.Decide(ctx, messages, service.DashScopeDecideQuestion{
		"need_decompose_tasks": service.DashScopeDecideQuestionContent{
			Type:         service.DashScopeDecideQuestionTypeNoul,
			Instructions: "Is user's latest message contain more than 1 tasks?",
			Criteria: service.DashScopeDecideQuestionCriteriaNoul{
				True:  "User's latest message contains more than 1 tasks.",
				False: "User's latest message contain only 1 task.",
			},
		},
	}, true)
	if err != nil {
		return false, err
	}
	// if has multiple tasks
	if needDecomposeTasks, ok := decideResponse.Answers["need_decompose_tasks"]; ok {
		noul, err := helper.ConvertJSON[service.DashScopeDecideResponseAnswerNoul](needDecomposeTasks)
		if err != nil {
			return false, fmt.Errorf("error parsing need_decompose_tasks to service.DashScopeDecideResponseAnswerNoul: %w", err)
		} else if noul.Noul > 0.5 {
			return true, nil
		}
	} else {
		return false, fmt.Errorf("error getting need_decompose_tasks from decideResponse.Answers")
	}
	return false, nil
}

func (chat *Chat) checkNeedRewrite(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) (bool, error) {
	// if there is only 1 user message, return false
	if helper.Count(messages, func(m service.DashScopeMessage) bool {
		return m.Role == service.DashScopeChatMessageRoleUser
	}) == 1 {
		return false, nil
	}
	decideResponse, err := dependencies.LLM.Decide(ctx, messages, service.DashScopeDecideQuestion{
		"need_rewrite": service.DashScopeDecideQuestionContent{
			Type:         service.DashScopeDecideQuestionTypeNoul,
			Instructions: "Is user's latest message contextual complete without rewriting needed?",
			Criteria: service.DashScopeDecideQuestionCriteriaNoul{
				True:  "User's latest message is contextual complete.",
				False: "User's latest message is not not contextual complete.",
			},
		},
	}, true)
	if err != nil {
		return false, err
	}
	// if need to rewrite
	if needRewrite, ok := decideResponse.Answers["need_rewrite"]; ok {
		noul, err := helper.ConvertJSON[service.DashScopeDecideResponseAnswerNoul](needRewrite)
		if err != nil {
			return false, fmt.Errorf("error parseing need_rewrite into service.DashScopeDecideResponseAnswerNoul: %w", err)
		}
		return noul.Noul > 0.5, nil
	} else {
		return false, fmt.Errorf("error getting answer from decide, need_rewrite is not found.")
	}
}

// task is a rewritten, single-task user query
func (chat *Chat) executeTask(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) (string, error) {
	skills, err := chat.detectSkills(ctx, messages, dependencies)
	if err != nil {
		return "", fmt.Errorf("error executing task in chat: %w", err)
	}
	// check if the top skill is RAG or function
	if len(skills) > 0 {
		if skills[0].Type == types.AIAgentSkillTypeAnswerQuestion {
			// do semantic search
			// call RAG
			return "Here is the answer to your question: blah blah", nil
		} else if len(skills[0].Inputs) > 0 {
			// if the skill requires user input
			// get all required inputs
			requiredInputs := helper.Filter(skills[0].Inputs, func(si dto_ai_agent.SkillInput) bool {
				return si.Required
			})
			missingInputs, err := chat.getMissingInputs(ctx, requiredInputs, messages, dependencies)
			if err != nil {
				return "", fmt.Errorf("error getting missing inputs: %w", err)
			}
			if len(missingInputs) == 0 {
				// all are provided, return friendly acknowledgement
				return "Thank you for your inputs, someone will get back to you soonest possible!", nil
			} else if len(missingInputs) == len(requiredInputs) {
				// none is provided, use LLM to ask user to provide
				// tell user what are the inputs required
				var inputTitles []string
				for i, input := range skills[0].Inputs {
					if input.Required {
						inputTitles = append(inputTitles, fmt.Sprintf("%d. %s (required)", i+1, input.Title))
					} else {
						inputTitles = append(inputTitles, fmt.Sprintf("%d. %s", i+1, input.Title))
					}
				}
				message := fmt.Sprintf("Thank you for your interest! In order to proceed, please provide the following details:\n\n%s", strings.Join(inputTitles, "\n"))
				return message, nil
			} else {
				// some are provided, some are missing, tell user to complete
				var missingInputTitles []string
				for i, input := range missingInputs {
					missingInputTitles = append(missingInputTitles, fmt.Sprintf("%d. %s (required)", i+1, input.Title))
				}
				message := fmt.Sprintf("Please also provide the following inputs in order to proceed, thank you!\n\n%s", strings.Join(missingInputTitles, "\n"))
				return message, nil
			}
		}
	}
	// reply
	reply := Reply{
		Messages:  chat.Messages,
		ProfileId: chat.Profile.Id,
		Skills:    skills,
	}
	replyResponse := reply.Handle(ctx, nil, dependencies)
	if !replyResponse.Success {
		return "", replyResponse.Error
	}
	return replyResponse.Data, nil
}

func (chat *Chat) getMissingInputs(ctx context.Context, skillInputs []dto_ai_agent.SkillInput, messages []service.DashScopeMessage, dependencies *service.Dependencies) ([]dto_ai_agent.SkillInput, error) {
	var questions service.DashScopeDecideQuestion = make(service.DashScopeDecideQuestion)
	for _, input := range skillInputs {
		if input.Required {
			questionContent := service.DashScopeDecideQuestionContent{
				Type: service.DashScopeDecideQuestionTypeNoul,
				Criteria: service.DashScopeDecideQuestionCriteriaNoul{
					True:  fmt.Sprintf("Yes, user has provided %s.", input.Title),
					False: fmt.Sprintf("No, user has not provided %s.", input.Title),
				},
			}
			questionContent.Instructions = fmt.Sprintf("Has the user provided this input in the conversation: %s?", input.Title)
			questions[input.Title] = questionContent
		}
	}
	response, err := dependencies.LLM.Decide(ctx, messages, questions, true)
	if err != nil {
		return nil, fmt.Errorf("error calling getMissingInputs to get skill inputs: %w", err)
	}
	var missingInputs []dto_ai_agent.SkillInput
	for _, input := range skillInputs {
		if !input.Required {
			continue
		}
		if result, ok := response.Answers[input.Title]; ok {
			noul, err := helper.ConvertJSON[service.DashScopeDecideResponseAnswerNoul](result)
			if err != nil {
				return nil, fmt.Errorf("error parsing response answer choide: %w", err)
			}
			// < 0.5 means user did not provide
			if noul.Noul < 0.5 {
				input := helper.First(skillInputs, func(si dto_ai_agent.SkillInput) bool {
					return si.Title == input.Title
				})
				missingInputs = append(missingInputs, *input)
			}
		} else {
			return nil, fmt.Errorf("%s is missing from getMissingInputs response answers", input.Title)
		}
	}
	return missingInputs, nil
}

func (chat *Chat) returnGenericResponse(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) (string, error) {
	return "Sorry, I am unable to assist.", nil
}
