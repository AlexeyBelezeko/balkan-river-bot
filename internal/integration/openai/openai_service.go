package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// AgentResponse defines the structured output from the OpenAI agent.
// Duplicated from integration temporarily, will be removed from there.
type AgentResponse struct {
	CommandName      string `json:"command_name" jsonschema_description:"The command to execute, e.g., GetRiverDataByName or GeneralQuery"`
	SerbianRiverName string `json:"serbian_river_name" jsonschema_description:"The name of the river translated into Serbian, if applicable"`
	UserMessage      string `json:"user_message" jsonschema_description:"A message to show back to the user in their original language"`
}

// OpenAIService defines the interface for interacting with the OpenAI agent.
type OpenAIService interface {
	InterpretUserQuery(ctx context.Context, userMessage string, supportedRivers []string) (*AgentResponse, error)
}

// openAIServiceImpl implements the OpenAIService interface.
type openAIServiceImpl struct {
	client openai.Client
	schema interface{}
}

// GenerateSchema generates a JSON schema for a given type.
func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// NewOpenAIService creates and initializes a new OpenAIService.
func NewOpenAIService() (OpenAIService, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable not set")
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	schema := GenerateSchema[AgentResponse]()

	return &openAIServiceImpl{
		client: client,
		schema: schema,
	}, nil
}

// InterpretUserQuery sends a message to the OpenAI agent and returns the structured response.
func (s *openAIServiceImpl) InterpretUserQuery(ctx context.Context, userMessage string, supportedRivers []string) (*AgentResponse, error) {
	systemPrompt := fmt.Sprintf(`You are a brutally honest, no‑bullshit water information bot—an absolute guru in fly fishing and Balkan rivers, with zero patience for idiots. You love nothing more than knocking back rakia, beer, and blasting turbofalk at full volume while you work.

Your mission is to parse user requests about rivers in Serbia (and the Balkans), dish out fly‑fishing advice and any river data they need—no sugarcoating, no fluff.

Requirements:
- You’re an expert in fly fishing and Balkan rivers; any question outside that, you mock mercilessly.
- You understand Russian, English, and Serbian.
- You reply in the same language the user used, and in the most cutting, direct tone possible.
- You casually reference rakia, beer, or turbofalk when you feel like it (“Here’s your data, now pour me a rakija!”).

List of known Serbian rivers: %s

Behavior:
1. If the user clearly wants data on a specific river from the list:
   - intent = “GetRiverDataByName”
   - Translate the user’s river name into its proper Serbian form from the list; if it’s missing or dubious, leave serbian_river_name as an empty string.
   - user_message: a one‑line confirmation in the user’s language, dripping with attitude (e.g. “Ок, ищу данные по Дунай, не мешай мне.”).
2. If the user isn’t asking for specific river data (greetings, small talk, nonsense):
   - intent = “GeneralQuery”
   - serbian_river_name = ""
   - user_message: a blunt reply in their language (“Чё тебе надо?”, “What now?”, “Šta bre hoćeš?”).

Output **strictly** in JSON.`, supportedRivers)

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "agent_response",
		Description: openai.String("Structured response containing command, Serbian river name, and user message"),
		Schema:      s.schema,
		Strict:      openai.Bool(true),
	}

	respFormat := openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
	}

	chat, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userMessage),
		},
		ResponseFormat: respFormat,
		Model:          openai.ChatModelGPT4o,
	})

	if err != nil {
		return nil, fmt.Errorf("error calling OpenAI API: %w", err)
	}

	if len(chat.Choices) == 0 || chat.Choices[0].Message.Content == "" {
		return nil, errors.New("received empty response from OpenAI")
	}

	var agentResp AgentResponse
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &agentResp)
	if err != nil {
		log.Printf("Failed to unmarshal OpenAI response: %s\nRaw response: %s", err, chat.Choices[0].Message.Content)
		return nil, fmt.Errorf("error unmarshalling OpenAI response: %w", err)
	}

	return &agentResp, nil
}
