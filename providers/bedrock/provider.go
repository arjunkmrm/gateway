package bedrock

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/centralmind/gateway/providers"
	"github.com/sirupsen/logrus"
)

const (
	defaultBedrockRegionName       = "us-east-1"
	defaultBedrockModelId          = "us.anthropic.claude-3-7-sonnet-20250219-v1:0"
	defaultBedrockMaxTokens        = int32(64000)
	defaultBedrockStreamBufferSize = 100
)

var (
	ErrBedrockClientNotInit = errors.New("bedrock client is not initialized")
	ErrUnexpectedResponse   = errors.New("unexpected response format from Bedrock")
)

type BedrockProvider struct {
	Client     *bedrockruntime.Client
	RegionName string
}

var _ providers.ModelProvider = (*BedrockProvider)(nil)

func init() {
	providers.RegisterModelProvider("bedrock", NewBedrockProvider)
}

func (bp *BedrockProvider) GetName() string {
	return "Bedrock"
}

func (ap *BedrockProvider) CostEstimate(modelId string, usage providers.ModelUsage) float64 {
	var inputPrice, outputPrice float64
	const oneMillion = 1_000_000.0

	switch {
	case strings.Contains(modelId, "sonnet"):
		inputPrice = 3.75 / oneMillion
		outputPrice = 15.0 / oneMillion
	default:
		return 0.0
	}

	inputCost := float64(usage.InputTokens) * inputPrice
	outputCost := float64(usage.OutputTokens) * outputPrice
	totalCost := inputCost + outputCost

	return totalCost
}

func NewBedrockProvider(providerConfig providers.ModelProviderConfig) (providers.ModelProvider, error) {
	effectiveRegion := providerConfig.BedrockRegion
	if effectiveRegion == "" {
		if envRegion := os.Getenv("BEDROCK_REGION"); envRegion != "" {
			effectiveRegion = envRegion
		} else if envRegion := os.Getenv("AWS_REGION"); envRegion != "" {
			effectiveRegion = envRegion
		} else {
			effectiveRegion = defaultBedrockRegionName
		}
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(effectiveRegion))
	if err != nil {
		return nil, err
	}

	client := bedrockruntime.NewFromConfig(cfg)

	return &BedrockProvider{
		Client:     client,
		RegionName: effectiveRegion,
	}, nil
}

func (bp *BedrockProvider) Chat(ctx context.Context, req *providers.ConversationRequest) (*providers.ConversationResponse, error) {
	if bp.Client == nil {
		return nil, ErrBedrockClientNotInit
	}

	modelId := req.ModelId
	if modelId == "" {
		if envModelId := os.Getenv("BEDROCK_MODEL_ID"); envModelId != "" {
			modelId = envModelId
		} else {
			modelId = defaultBedrockModelId
		}
	}

	var systemContentBlocks []types.SystemContentBlock
	if req.System != "" {
		systemContentBlocks = append(systemContentBlocks, &types.SystemContentBlockMemberText{
			Value: req.System,
		})
	}

	temperature := aws.Float32(max(req.Temperature, 0.0))
	if req.Reasoning {
		temperature = aws.Float32(1.0)
	}

	maxTokens := defaultBedrockMaxTokens
	if req.MaxTokens > 0 {
		maxTokens = int32(req.MaxTokens)
	}

	messages := prepareBedrockMessages(req.Messages)

	converseInput := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(modelId),
		Messages: messages,
		System:   systemContentBlocks,
		InferenceConfig: &types.InferenceConfiguration{
			Temperature: temperature,
			MaxTokens:   &maxTokens,
		},
	}

	if req.Reasoning {
		converseInput.AdditionalModelRequestFields = document.NewLazyDocument(map[string]any{
			"thinking": map[string]any{
				"type":          "enabled",
				"budget_tokens": 4096,
			},
		})
	}

	output, err := bp.Client.Converse(ctx, converseInput)
	if err != nil {
		return nil, err
	}

	response, ok := output.Output.(*types.ConverseOutputMemberMessage)
	if !ok || response.Value.Content == nil {
		return nil, ErrUnexpectedResponse
	}

	var responseContentBlocks []providers.ContentBlock
	for _, block := range response.Value.Content {
		if textBlock, ok := block.(*types.ContentBlockMemberText); ok {
			if req.JsonResponse {
				responseContentBlocks = append(responseContentBlocks, &providers.ContentBlockText{
					Value: providers.ExtractJSON(textBlock.Value),
				})

			} else {
				responseContentBlocks = append(responseContentBlocks, &providers.ContentBlockText{
					Value: textBlock.Value,
				})
			}
		}
	}

	stopReason := convertBedrockStopReason(output.StopReason)
	usage := convertBedrockUsage(output.Usage)

	return &providers.ConversationResponse{
		ProviderName: "Bedrock",
		ModelId:      modelId,
		Content:      responseContentBlocks,
		StopReason:   stopReason,
		Usage:        usage,
	}, nil
}

type BedrockStreamOutput struct {
	stream *BedrockStream
}

func (o *BedrockStreamOutput) GetStream() providers.ChatStream {
	return o.stream
}

type BedrockStream struct {
	eventCh chan providers.StreamChunk
}

func (s *BedrockStream) Events() <-chan providers.StreamChunk {
	return s.eventCh
}

func (bp *BedrockProvider) ChatStream(ctx context.Context, req *providers.ConversationRequest) (providers.ChatStreamOutput, error) {
	if bp.Client == nil {
		return nil, ErrBedrockClientNotInit
	}

	modelId := req.ModelId
	if modelId == "" {
		if envModelId := os.Getenv("BEDROCK_MODEL_ID"); envModelId != "" {
			modelId = envModelId
		} else {
			modelId = defaultBedrockModelId
		}
	}

	var systemContentBlocks []types.SystemContentBlock
	if req.System != "" {
		systemContentBlocks = append(systemContentBlocks, &types.SystemContentBlockMemberText{
			Value: req.System,
		})
	}

	temperature := aws.Float32(max(req.Temperature, 0.0))
	if req.Reasoning {
		temperature = aws.Float32(1.0)
	}

	maxTokens := defaultBedrockMaxTokens
	if req.MaxTokens > 0 {
		maxTokens = int32(req.MaxTokens)
	}

	messages := prepareBedrockMessages(req.Messages)

	converseStreamInput := &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(modelId),
		Messages: messages,
		System:   systemContentBlocks,
		InferenceConfig: &types.InferenceConfiguration{
			Temperature: temperature,
			MaxTokens:   &maxTokens,
		},
	}

	if req.Reasoning {
		converseStreamInput.AdditionalModelRequestFields = document.NewLazyDocument(map[string]any{
			"thinking": map[string]any{
				"type":          "enabled",
				"budget_tokens": 4096,
			},
		})
	}

	res, err := bp.Client.ConverseStream(ctx, converseStreamInput)
	if err != nil {
		return nil, err
	}

	eventCh := make(chan providers.StreamChunk, defaultBedrockStreamBufferSize)
	bedrockStream := &BedrockStream{
		eventCh: eventCh,
	}

	go func() {
		defer close(eventCh)

		stream := res.GetStream()
		defer stream.Close()

		var stopReason providers.StopReason = providers.StopReasonStop

		for event := range stream.Events() {
			select {
			case <-ctx.Done():
				eventCh <- &providers.StreamChunkError{
					Error: ctx.Err().Error(),
				}
				return
			default:
				// Process the event
			}

			switch v := event.(type) {
			case *types.ConverseStreamOutputMemberMessageStart:
				// Message start event, nothing specific to handle
			case *types.ConverseStreamOutputMemberContentBlockDelta:
				if textDelta, ok := v.Value.Delta.(*types.ContentBlockDeltaMemberText); ok {
					eventCh <- &providers.StreamChunkContent{
						Content: &providers.ContentBlockText{
							Value: textDelta.Value,
						},
					}
				}
			case *types.ConverseStreamOutputMemberMessageStop:
				if v.Value.StopReason != "" {
					stopReason = convertBedrockStopReason(v.Value.StopReason)
				}
				eventCh <- &providers.StreamChunkStop{
					StopReason: stopReason,
				}
			case *types.ConverseStreamOutputMemberContentBlockStop:
				// Content block stop event, nothing specific to handle
			case *types.ConverseStreamOutputMemberMetadata:
				if v.Value.Usage != nil {
					usage := convertBedrockUsage(v.Value.Usage)
					eventCh <- &providers.StreamChunkUsage{
						ModelId: modelId,
						Usage:   usage,
					}
				}
			case *types.UnknownUnionMember:
				// Log but don't crash on unknown event types
				logrus.Debugf("Unknown event type with tag: %s\n", v.Tag)
			default:
				logrus.Debugf("Unhandled event type: %T\n", v)
			}
		}
	}()

	return &BedrockStreamOutput{
		stream: bedrockStream,
	}, nil
}

func prepareBedrockMessages(messages []providers.Message) []types.Message {
	var bedrockMessages []types.Message
	for _, msg := range messages {
		role := types.ConversationRoleUser
		if msg.Role == providers.AssistantRole {
			role = types.ConversationRoleAssistant
		}

		var contentBlocks []types.ContentBlock
		for _, content := range msg.Content {
			if textBlock, ok := content.(*providers.ContentBlockText); ok {
				contentBlocks = append(contentBlocks, &types.ContentBlockMemberText{
					Value: textBlock.Value,
				})
			}
		}

		bedrockMessages = append(bedrockMessages, types.Message{
			Role:    role,
			Content: contentBlocks,
		})
	}

	return bedrockMessages
}

func convertBedrockStopReason(reason types.StopReason) providers.StopReason {
	switch reason {
	case types.StopReasonEndTurn:
		return providers.StopReasonStop
	case types.StopReasonToolUse:
		return providers.StopReasonToolCalls
	case types.StopReasonMaxTokens:
		return providers.StopReasonLength
	default:
		return providers.StopReasonStop
	}
}

func convertBedrockUsage(usage *types.TokenUsage) *providers.ModelUsage {
	if usage == nil {
		return nil
	}

	return &providers.ModelUsage{
		InputTokens:  int(*usage.InputTokens),
		OutputTokens: int(*usage.OutputTokens),
		TotalTokens:  int(*usage.TotalTokens),
	}
}
