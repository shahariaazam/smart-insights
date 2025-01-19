package models

import "time"

type AssistantRequestOptions struct {
	LLMProvider string `json:"llm_provider" validate:"required,oneof=openai anthropic gemini bedrock"`
	LLMConfig   string `json:"llm_config" validate:"required"`
}

type AssistantRequest struct {
	DBConfigurationName string                  `json:"db_configuration_name" validate:"required"`
	Question            string                  `json:"question" validate:"required"`
	Options             AssistantRequestOptions `json:"options" validate:"required"`
}

type AssistantResponse struct {
	UUID     string   `json:"uuid"`
	Question string   `json:"question"`
	Success  bool     `json:"success"`
	Status   string   `json:"status"`
	Response []Update `json:"response,omitempty"`
}

type Update struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}
