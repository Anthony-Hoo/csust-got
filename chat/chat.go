package chat

import (
	"context"
	"csust-got/config"
	"csust-got/entities"
	"csust-got/log"
	"csust-got/orm"
	"csust-got/util"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	. "gopkg.in/telebot.v3"
)

var (
	client   *openai.Client
	chatChan = make(chan *chatContext, 16)
)

type chatContext struct {
	Context
	req *openai.ChatCompletionRequest
	msg *Message
}

// InitChat init chat service
func InitChat() {
	if config.BotConfig.ChatConfig.Key != "" {
		client = openai.NewClient(config.BotConfig.ChatConfig.Key)
		go chatService()
	}
}

// GPTChat is handler for chat with GPT
func GPTChat(ctx Context) error {
	return chat(ctx, false)
}

// GPTChatWithStream is handler for chat with GPT, and use stream api
func GPTChatWithStream(ctx Context) error {
	return chat(ctx, true)
}

func chat(ctx Context, stream bool) error {
	if client == nil {
		return nil
	}

	_, arg, err := entities.CommandTakeArgs(ctx.Message(), 0)
	if err != nil {
		log.Error("[ChatGPT] Can't take args", zap.Error(err))
		return ctx.Reply("嗦啥呢？")
	}
	if len(arg) == 0 {
		return ctx.Reply("您好，有什么问题可以为您解答吗？")
	}
	if len(arg) > config.BotConfig.ChatConfig.PromptLimit {
		return ctx.Reply("TLDR")
	}

	req, err := generateRequest(ctx, arg, stream)
	if err != nil {
		return err
	}

	msg, err := util.SendReplyWithError(ctx.Chat(), "正在思考...", ctx.Message())
	if err != nil {
		return err
	}

	payload := &chatContext{Context: ctx, req: req, msg: msg}

	select {
	case chatChan <- payload:
		return nil
	default:
		return ctx.Reply("要处理的对话太多了，要不您稍后再试试？")
	}
}

func generateRequest(ctx Context, arg string, stream bool) (*openai.ChatCompletionRequest, error) {
	chatCfg := config.BotConfig.ChatConfig
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT3Dot5Turbo,
		MaxTokens:   chatCfg.MaxTokens,
		Messages:    []openai.ChatCompletionMessage{},
		Stream:      stream,
		Temperature: chatCfg.Temperature,
	}

	if chatCfg.Model != "" {
		req.Model = chatCfg.Model
	}

	if len(req.Messages) == 0 && chatCfg.SystemPrompt != "" {
		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: chatCfg.SystemPrompt,
		})
	}

	keepContext := chatCfg.KeepContext
	if keepContext > 0 && ctx.Message().ReplyTo != nil {
		chatContext, err := orm.GetChatContext(ctx.Chat().ID, ctx.Message().ReplyTo.ID)
		if err == nil {
			if len(chatContext) > 2*keepContext {
				chatContext = chatContext[len(chatContext)-2*keepContext:]
			}
			req.Messages = append(req.Messages, chatContext...)
		}
	}

	req.Messages = append(req.Messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: arg})

	return &req, nil
}

func chatService() {
	for ctx := range chatChan {
		go func(ctx *chatContext) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("[ChatGPT] Panic", zap.Any("err", err))
				}
			}()

			if ctx.req.Stream {
				chatWithStream(ctx)
			} else {
				chatWithoutStream(ctx)
			}

		}(ctx)
	}
}

func chatWithoutStream(ctx *chatContext) {
	start := time.Now()

	resp, err := client.CreateChatCompletion(context.Background(), *ctx.req)
	if err != nil {
		log.Error("[ChatGPT] Can't create completion", zap.Error(err))
		return
	}

	content := resp.Choices[0].Message.Content

	if strings.TrimSpace(content) == "" {
		content += "\n...嗦不粗话"
	}

	if config.BotConfig.DebugMode {
		content += fmt.Sprintf("\n\nusage: %d + %d = %d\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		content += fmt.Sprintf("time cost: %v\n", time.Since(start))
	}
	replyMsg, err := util.EditMessageWithError(ctx.msg, content)
	if err != nil {
		log.Error("[ChatGPT] Can't edit message", zap.Error(err))
		return
	}

	err = orm.SetChatContext(ctx.Context.Chat().ID, replyMsg.ID, ctx.req.Messages)
	if err != nil {
		log.Error("[ChatGPT] Can't set chat context", zap.Error(err))
	}

}

func chatWithStream(ctx *chatContext) {
	start := time.Now()

	var replyMsg *Message

	stream, err := client.CreateChatCompletionStream(context.Background(), *ctx.req)
	if err != nil {
		_, err = util.EditMessageWithError(ctx.msg,
			"An error occurred. If this issue persists please contact us through our help center at help.openai.com.")
		if err != nil {
			log.Error("[ChatGPT] Can't edit message", zap.Error(err))
		}
		return
	}
	defer stream.Close()

	content := ""
	contentLock := sync.Mutex{}
	done := make(chan struct{})
	go func() {
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				ctx.req.Messages = append(ctx.req.Messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: content,
				})
				done <- struct{}{}
				break
			}

			if err != nil {
				contentLock.Lock()
				content += "\n\n...寄了"
				contentLock.Unlock()
				log.Error("[ChatGPT] Stream error", zap.Error(err))
				break
			}

			contentLock.Lock()
			content += response.Choices[0].Delta.Content
			contentLock.Unlock()
		}
	}()

	ticker := time.NewTicker(2 * time.Second) // 编辑过快会被tg限流
	defer ticker.Stop()
	lastContent := "" // 记录上次编辑的内容，内容相同则不再编辑，避免tg的api返回400
out:
	for range ticker.C {
		contentLock.Lock()
		contentCopy := content
		contentLock.Unlock()
		if len(strings.TrimSpace(contentCopy)) > 0 && strings.TrimSpace(contentCopy) != strings.TrimSpace(lastContent) {
			replyMsg, err = util.EditMessageWithError(ctx.msg, contentCopy)
			if err != nil {
				log.Error("[ChatGPT] Can't edit message", zap.Error(err))
			} else {
				lastContent = contentCopy
			}
		}
		select {
		case <-done:
			break out
		default:
		}
	}

	contentLock.Lock()
	if strings.TrimSpace(content) == "" {
		content += "\n...嗦不粗话"
	}
	if config.BotConfig.DebugMode {
		content += fmt.Sprintf("\n\ntime cost: %v\n", time.Since(start))
		replyMsg, err = util.EditMessageWithError(ctx.msg, content)
		if err != nil {
			log.Error("[ChatGPT] Can't edit message", zap.Error(err))
		}
	}
	contentLock.Unlock()

	if replyMsg != nil {
		err = orm.SetChatContext(ctx.Context.Chat().ID, replyMsg.ID, ctx.req.Messages)
		if err != nil {
			log.Error("[ChatGPT] Can't set chat context", zap.Error(err))
		}
	}
}