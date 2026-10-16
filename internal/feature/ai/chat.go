package feture_ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jjcheng/wawa-go/internal/dto"
	dto_ai "github.com/jjcheng/wawa-go/internal/dto/ai"
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

func (chat Chat) Handle(ctx context.Context, dependencies *service.Dependencies) dto.Response[*dto_ai.ChatResponse] {
	if errors := chat.Validate(); len(errors) > 0 {
		return dto.NewInvalidInputResponse[*dto_ai.ChatResponse](errors)
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
	invalid, err := chat.checkIsInvalid(ctx, serviceMessages, dependencies)
	if err != nil {
		return dto.NewFailedResponse[*dto_ai.ChatResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
	}
	if invalid {
		return dto.NewSuccessResponse[*dto_ai.ChatResponse](nil)
	}
	// step 2: rewrite to decompose tasks
	rewrite := Rewrite{
		Messages: chat.Messages,
	}
	rewriteResponse := rewrite.Handle(ctx, dependencies)
	if !rewriteResponse.Success {
		chatResponse := dto_ai.NewChatResponse(types.AIErrorResponse, nil)
		return dto.NewSuccessResponse(&chatResponse)
	}
	// step 3: execute each task parallelly
	newData := []string{}
	nextSteps := []dto_ai.ChatNextStep{}
	for _, task := range rewriteResponse.Data {
		message, nextStep, err := chat.executeTask(ctx, serviceMessages, task, dependencies)
		if err != nil {
			return dto.NewFailedResponse[*dto_ai.ChatResponse](http.StatusInternalServerError, types.ExceptionMessageInternalServerError, err)
		}
		if nextStep != nil {
			nextSteps = append(nextSteps, *nextStep)
		}
		newData = append(newData, message)
	}
	// step 4: if there is more than 1 task, join the data from tasks, reply
	var chatResponse dto_ai.ChatResponse
	if len(rewriteResponse.Data) > 1 {
		reply := Reply{
			Messages:       chat.Messages,
			Profile:        chat.Profile,
			AdditionalData: strings.Join(newData, "\n---\n"),
		}
		replyResponse := reply.Handle(ctx, dependencies)
		if !replyResponse.Success {
			return dto.NewFailedResponse[*dto_ai.ChatResponse](replyResponse.StatusCode, replyResponse.Message, replyResponse.Error)
		}
		chatResponse = dto_ai.NewChatResponse(replyResponse.Data, nil)
	} else {
		chatResponse = dto_ai.NewChatResponse(newData[0], nil)
	}
	if len(nextSteps) > 0 {
		chatResponse.NextStep = &nextSteps[len(nextSteps)-1]
	}
	return dto.NewSuccessResponse(&chatResponse)
}

// check if the user query is spam, scam, nonsense or inapproriate
func (chat *Chat) checkIsInvalid(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) (bool, error) {
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

// task is a rewritten, single self contained user query
func (chat *Chat) executeTask(ctx context.Context, messages []service.DashScopeMessage, task string, dependencies *service.Dependencies) (string, *dto_ai.ChatNextStep, error) {
	skills, err := chat.decideSkills(ctx, messages, dependencies)
	if err != nil {
		return "", nil, fmt.Errorf("error deciding skills when executing task: %w", err)
	}
	// check if the top skill is RAG or function
	if len(skills) > 0 {
		if skills[0].Type == types.AIAgentSkillTypeAnswerQuestion { // rag
			message, err := chat.rag(ctx, skills, task, dependencies)
			if err != nil {
				return "", nil, err
			}
			return message, nil, nil
		} else if len(skills[0].Inputs) > 0 { // function call
			return chat.callFunction(ctx, skills, messages, dependencies)
		}
	}
	// reply
	reply, err := chat.reply(ctx, skills, "", dependencies)
	return reply, nil, err
}

// find out what are the skills to be applied to reply
// always_apply skills are always included
func (chat *Chat) decideSkills(ctx context.Context, messages []service.DashScopeMessage, dependencies *service.Dependencies) ([]dto_ai_agent.Skill, error) {
	listSkills := feature_ai_agent_skill.List{
		ProfileId:   chat.Profile.Id,
		FromWebhook: true,
		OnlyEnabled: true,
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
	// add a non of above skill as recommended
	criteria = append(criteria, "None of above")
	skillScores, err := dependencies.LLM.Decide(ctx, messages, service.DashScopeDecideQuestion{
		"detect_skills": service.DashScopeDecideQuestionContent{
			Type:         service.DashScopeDecideQuestionTypeScore,
			Instructions: "Which of the criteria should be applied when replying to user's latest message?",
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
		type detectedSkill struct {
			skillTitle string
			score      float32
		}
		var orderedSkills []detectedSkill
		for k, v := range scores.Probabilities {
			if v < chat.MinSkillScore {
				match := helper.First(listSkillsResponse.Data, func(s dto_ai_agent.Skill) bool {
					return s.Title == scores.Legend[k]
				})
				if match == nil || !match.AlwaysApply {
					continue
				}
			}
			orderedSkills = append(orderedSkills, detectedSkill{
				skillTitle: scores.Legend[k],
				score:      v,
			})
		}
		helper.Sort(orderedSkills, func(a, b detectedSkill) int {
			if a.score > b.score {
				return -1
			} else if a.score < b.score {
				return 1
			} else {
				return 0
			}
		})
		skills := helper.Map(orderedSkills, func(t detectedSkill) dto_ai_agent.Skill {
			match := helper.First(listSkillsResponse.Data, func(s dto_ai_agent.Skill) bool {
				return s.Title == t.skillTitle
			})
			return *match
		})
		return skills, nil
	} else {
		return nil, fmt.Errorf("detect skills not found in detectSkill function")
	}
}

// do semantic search, then do rag
func (chat *Chat) rag(ctx context.Context, skills []dto_ai_agent.Skill, task string, dependencies *service.Dependencies) (string, error) {
	// do semantic search
	searchKnowledge := Search{
		Query:     task,
		ProfileId: chat.Profile.Id,
		Semantic:  false,
	}
	searchKnowledgesResponse := searchKnowledge.Handle(ctx, dependencies)
	if !searchKnowledgesResponse.Success {
		return "", searchKnowledgesResponse.Error
	}
	// call RAG
	if len(searchKnowledgesResponse.Data) > 0 {
		rag := RAG{
			Profile:    chat.Profile,
			Query:      task,
			Knowledges: searchKnowledgesResponse.Data,
			Skills:     skills,
		}
		ragResponse := rag.Handle(ctx, dependencies)
		if !ragResponse.Success {
			return types.AIErrorResponse, nil
		}
		return ragResponse.Data, nil
	} else {
		return chat.reply(ctx, skills[1:], "User asked a question we cannot answer. Give a polite sorry message to inform user that we could not find any relevant data and ask whatelse we can help.", dependencies)
	}
}

// gather required input and send notification
func (chat *Chat) callFunction(ctx context.Context, skills []dto_ai_agent.Skill, messages []service.DashScopeMessage, dependencies *service.Dependencies) (string, *dto_ai.ChatNextStep, error) {
	// get all required inputs
	requiredInputs := helper.Filter(skills[0].Inputs, func(si dto_ai_agent.SkillInput) bool {
		return si.Required
	})
	// decide missing inputs
	missingInputs, err := chat.decideMissingInputs(ctx, requiredInputs, messages, dependencies)
	if err != nil {
		return "", nil, fmt.Errorf("error deciding missing inputs when executing task: %w", err)
	}
	if len(missingInputs) == 0 { // all inputs are provided, reply friendly acknowledgement
		replyRepsonse, err := chat.reply(ctx, skills, fmt.Sprintf("The process of %s is completed, give a friendly acknowledgement and ask whatelse we can help. Do NOT request for more information.", skills[0].Title), dependencies)
		if err != nil {
			return "", nil, err
		}
		last15Messages := messages
		if len(last15Messages) > 15 {
			last15Messages = last15Messages[len(last15Messages)-15:]
		}
		body := []string{
			"Most recent 15 messages:",
		}
		for _, message := range last15Messages {
			if message.Role == service.DashScopeChatMessageRoleUser {
				body = append(body, fmt.Sprintf("User:\n%s", message.Content))
			} else {
				body = append(body, fmt.Sprintf("AI:\n%s", message.Content))
			}
		}
		nextStep := dto_ai.NewChatNextStep(types.AIChatNextStepTypeNotifyUser, fmt.Sprintf("User completed skill: %s", skills[0].Title), strings.Join(body, "\n\n"))
		return replyRepsonse, &nextStep, nil
	} else if len(missingInputs) == len(requiredInputs) {
		// tell user what are the inputs required
		var inputTitles []string
		for i, input := range skills[0].Inputs {
			if input.Required {
				inputTitles = append(inputTitles, fmt.Sprintf("%d. %s (required)", i+1, input.Title))
			} else {
				inputTitles = append(inputTitles, fmt.Sprintf("%d. %s", i+1, input.Title))
			}
		}
		reply, err := chat.reply(ctx, skills, fmt.Sprintf("To do %s, we need following inputs from the user:\n%s\n\nPolitely ask user to provide them.", skills[0].Title, strings.Join(inputTitles, "\n")), dependencies)
		return reply, nil, err
	} else {
		// some are provided, some are missing, tell user to complete
		var missingInputTitles []string
		for i, input := range missingInputs {
			missingInputTitles = append(missingInputTitles, fmt.Sprintf("%d. %s (required)", i+1, input.Title))
		}
		reply, err := chat.reply(ctx, skills, fmt.Sprintf("User has provided some inputs but some are still missing. Give a polite message to request for following:\n%s", strings.Join(missingInputTitles, "\n")), dependencies)
		return reply, nil, err
	}
}

func (chat *Chat) reply(ctx context.Context, skills []dto_ai_agent.Skill, instruction string, dependencies *service.Dependencies) (string, error) {
	reply := Reply{
		Messages:    chat.Messages,
		Profile:     chat.Profile,
		Skills:      skills,
		Instruction: instruction,
	}
	replyResponse := reply.Handle(ctx, dependencies)
	if !replyResponse.Success {
		return "", replyResponse.Error
	}
	return replyResponse.Data, nil
}

// detect if each of the required inputs are provided
func (chat *Chat) decideMissingInputs(ctx context.Context, skillInputs []dto_ai_agent.SkillInput, messages []service.DashScopeMessage, dependencies *service.Dependencies) ([]dto_ai_agent.SkillInput, error) {
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
