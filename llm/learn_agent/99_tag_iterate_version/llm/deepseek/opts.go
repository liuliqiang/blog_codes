package deepseek

import agentloop "github.com/liuliqiang/llmagent/99_tag_iterate_version"

type DeepseekClientOptions interface {
	WithModel(model agentloop.Model) DeepseekClientOptions
	WithAnthropicBaseURL(url string) DeepseekClientOptions
	WithResponseBaseURL(url string) DeepseekClientOptions
	WithAPIKey(apiKey string) DeepseekClientOptions
}

type deepseekClientOptions struct {
	model           agentloop.Model
	anthopicBaseURL string
	responseBaseURL string
	apiKey          string
}

func NewDeepseekClientOptions(model agentloop.Model) *deepseekClientOptions {
	return &deepseekClientOptions{
		model: model,
	}
}

func (o *deepseekClientOptions) WithAnthropicBaseURL(url string) DeepseekClientOptions {
	o.anthopicBaseURL = url
	return o
}

func (o *deepseekClientOptions) WithResponseBaseURL(url string) DeepseekClientOptions {
	o.responseBaseURL = url
	return o
}

func (o *deepseekClientOptions) WithAPIKey(apiKey string) DeepseekClientOptions {
	o.apiKey = apiKey
	return o
}

func (o *deepseekClientOptions) WithModel(model agentloop.Model) DeepseekClientOptions {
	o.model = model
	return o
}
