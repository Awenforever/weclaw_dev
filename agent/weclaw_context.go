package agent

import "strings"

const weClawCapabilitySystemPrompt = `WeClaw runtime context:
- You are operating inside WeClaw Dev, a WeChat-integrated agent environment. The current user message comes from the active WeChat chat, and phrases such as "send it to me", "send via WeChat", "发给我" or "通过微信发给我" refer to that current chat.
- WeClaw can send text replies and can send generated images/files back to the current WeChat chat when you provide a supported local attachment path.
- To send a generated image or file, save it under the current working directory, the configured WeClaw save directory, or ~/.weclaw/workspace, then output the absolute file path on a line by itself.
- Supported attachment types include png, jpg, jpeg, gif, webp, pdf, doc, docx, xls, xlsx, ppt, pptx, zip, txt, csv, mp4 and mov.
- Do not claim that you cannot send through WeChat when this WeClaw attachment workflow applies. If you output a standalone supported local path and WeClaw sends it successfully, treat the file as sent. Only give manual download or forwarding instructions when the user asks for them or the WeClaw send path is unavailable.`

// DefaultWeClawSystemPrompt returns the runtime capability contract that lets agents
// understand they are operating inside a WeChat-connected WeClaw environment.
func DefaultWeClawSystemPrompt() string {
	return weClawCapabilitySystemPrompt
}

// MergeWeClawSystemPrompt prepends the WeClaw runtime contract while preserving
// any user-configured system prompt from ~/.weclaw/config.json.
func MergeWeClawSystemPrompt(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return weClawCapabilitySystemPrompt
	}
	if strings.Contains(configured, "WeClaw runtime context:") {
		return configured
	}
	return weClawCapabilitySystemPrompt + "\n\nUser-configured agent system prompt:\n" + configured
}

// ComposeUserMessageWithSystemPrompt is used for agent protocols that do not
// expose a reliable native system/developer instruction channel. It keeps the
// user's message intact while adding the runtime contract above it.
func ComposeUserMessageWithSystemPrompt(systemPrompt, message string) string {
	systemPrompt = strings.TrimSpace(systemPrompt)
	if systemPrompt == "" {
		return message
	}
	return systemPrompt + "\n\n--- User message from the current WeChat chat ---\n" + message
}
