package oauthprovider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const (
	codexDefaultImageGenerationModel = "gpt-image-2"
	codexDefaultImageGenerationSize  = "1024x1024"
	maxImageGenerationResults        = 4
	maxImageGenerationSSEBytes       = 64 * 1024 * 1024
	maxImageGenerationEvents         = 512
)

func (p *CodexProvider) SupportsImageGeneration() bool {
	return true
}

func (p *CodexProvider) ImageGenerationProviderID() string {
	return "openai-codex"
}

func (p *CodexProvider) DefaultImageGenerationModel() string {
	return codexDefaultImageGenerationModel
}

func (p *CodexProvider) GenerateImage(
	ctx context.Context,
	req ImageGenerationRequest,
) (*ImageGenerationResponse, error) {
	opts, accountID, err := p.requestOptions()
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		return nil, fmt.Errorf("no account id found for Codex image generation")
	}

	if strings.TrimSpace(req.Model) == "" {
		req.Model = p.DefaultImageGenerationModel()
	}
	if req.Count < 1 {
		req.Count = 1
	}
	if req.Count > maxImageGenerationResults {
		req.Count = maxImageGenerationResults
	}

	images := make([]GeneratedImage, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		params := buildCodexImageParams(req)
		stream := p.client.Responses.NewStreaming(ctx, params, opts...)
		eventImages, readErr := parseCodexImageSSE(stream, req.OutputFormat)
		closeErr := stream.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		images = append(images, eventImages...)
	}
	if len(images) > maxImageGenerationResults {
		images = images[:maxImageGenerationResults]
	}
	return &ImageGenerationResponse{Images: images}, nil
}

func (p *CodexProvider) requestOptions() ([]option.RequestOption, string, error) {
	var opts []option.RequestOption
	accountID := p.accountID
	if p.tokenSource != nil {
		tok, accID, err := p.tokenSource()
		if err != nil {
			return nil, "", fmt.Errorf("refreshing token: %w", err)
		}
		opts = append(opts, option.WithAPIKey(tok))
		if accID != "" {
			accountID = accID
		}
	}
	if accountID != "" {
		opts = append(opts, option.WithHeader("Chatgpt-Account-Id", accountID))
	}
	return opts, accountID, nil
}

func buildCodexImageParams(req ImageGenerationRequest) responses.ResponseNewParams {
	size := strings.TrimSpace(req.Size)
	if size == "" {
		size = codexDefaultImageGenerationSize
	}

	tool := responses.ToolUnionParam{OfImageGeneration: &responses.ToolImageGenerationParam{
		Model: req.Model,
		Size:  size,
	}}
	if req.Quality != "" {
		tool.OfImageGeneration.Quality = req.Quality
	}
	if req.OutputFormat != "" {
		tool.OfImageGeneration.OutputFormat = req.OutputFormat
	}

	content := responses.ResponseInputMessageContentListParam{
		responses.ResponseInputContentParamOfInputText(req.Prompt),
	}
	input := responses.ResponseInputParam{
		responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRoleUser),
	}

	return responses.ResponseNewParams{
		Model: "gpt-5.4",
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
		Instructions: openai.Opt("You are an image generation assistant."),
		Tools:        []responses.ToolUnionParam{tool},
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfHostedTool: &responses.ToolChoiceTypesParam{
				Type: responses.ToolChoiceTypesTypeImageGeneration,
			},
		},
		Store: openai.Opt(false),
	}
}

type codexImageStream interface {
	Next() bool
	Current() responses.ResponseStreamEventUnion
	Err() error
}

func parseCodexImageSSE(stream codexImageStream, outputFormat string) ([]GeneratedImage, error) {
	var totalBytes int
	var events int
	var images []GeneratedImage
	var completedImages []GeneratedImage

	for stream.Next() {
		evt := stream.Current()
		events++
		if events > maxImageGenerationEvents {
			return nil, fmt.Errorf("codex image response exceeded event limit")
		}
		data, err := json.Marshal(evt)
		if err == nil {
			totalBytes += len(data)
			if totalBytes > maxImageGenerationSSEBytes {
				return nil, fmt.Errorf("codex image response exceeded size limit")
			}
		}
		eventImages, eventCompletedImages, parseErr := parseCodexImageEventUnion(evt, outputFormat)
		if parseErr != nil {
			return nil, parseErr
		}
		images = append(images, eventImages...)
		completedImages = append(completedImages, eventCompletedImages...)
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	if len(images) > 0 {
		return images, nil
	}
	return completedImages, nil
}

func parseCodexImageEventUnion(
	evt responses.ResponseStreamEventUnion,
	outputFormat string,
) ([]GeneratedImage, []GeneratedImage, error) {
	switch evt.Type {
	case "response.output_item.done":
		if image, ok, err := imageFromCodexItemUnion(evt.Item, outputFormat); err != nil {
			return nil, nil, err
		} else if ok {
			return []GeneratedImage{image}, nil, nil
		}
	case "response.completed":
		images := make([]GeneratedImage, 0)
		for _, item := range evt.Response.Output {
			if image, ok, err := imageFromCodexResponseItem(item, outputFormat); err != nil {
				return nil, nil, err
			} else if ok {
				images = append(images, image)
			}
		}
		return nil, images, nil
	case "response.failed", "error":
		return nil, nil, fmt.Errorf("codex image generation failed")
	}
	return nil, nil, nil
}

func imageFromCodexItemUnion(
	item responses.ResponseOutputItemUnion,
	outputFormat string,
) (GeneratedImage, bool, error) {
	if item.Type != "image_generation_call" {
		return GeneratedImage{}, false, nil
	}
	return imageFromCodexPayload(item.Result, outputFormat)
}

func imageFromCodexResponseItem(
	item responses.ResponseOutputItemUnion,
	outputFormat string,
) (GeneratedImage, bool, error) {
	return imageFromCodexItemUnion(item, outputFormat)
}

func imageFromCodexPayload(payload string, outputFormat string) (GeneratedImage, bool, error) {
	if payload == "" {
		return GeneratedImage{}, false, nil
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return GeneratedImage{}, false, err
	}
	mime, ext := imageMimeAndExtension(outputFormat)
	return GeneratedImage{Data: data, MimeType: mime, Ext: ext}, true, nil
}

func imageMimeAndExtension(outputFormat string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "jpeg", "jpg":
		return "image/jpeg", "jpg"
	case "webp":
		return "image/webp", "webp"
	default:
		return "image/png", "png"
	}
}
