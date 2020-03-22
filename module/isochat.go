package module

import (
	"csust-got/module/conds"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type Factory func(update tgbotapi.Update) Module

type isolatedChatModule struct {
	// Key: chat id
	// Value: handleModule
	registeredMods map[int64]Module
	factory        Factory
	shouldRegister conds.Predicate
}

func (i *isolatedChatModule) HandleUpdate(context Context, update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	chat := update.Message.Chat
	handler := i.registeredMods[chat.ID]
	handler.HandleUpdate(context.SubContext(fmt.Sprint(chat.ID)), update, bot)
}

func (i *isolatedChatModule) ShouldHandle(context Context, update tgbotapi.Update) bool {
	chat := update.Message.Chat
	// Registered chat.
	newCtx := context.SubContext(fmt.Sprint(chat.ID))
	if module, ok := i.registeredMods[chat.ID]; ok {
		return module.ShouldHandle(newCtx, update)
	}
	// Not yet registered chat, but we should register now.
	if i.shouldRegister.ShouldHandle(update) {
		module := i.factory(update)
		i.registeredMods[chat.ID] = module
		return module.ShouldHandle(newCtx, update)
	}
	return false
}

// IsolatedChat returns a Module that will, for each incoming update, split it by chatID.
// 对于每一个 update，IsolatedChat 将会借助 factory 创建单独的一个模块进行处理。
// 注意仅当 shouldRegister 返回为真的时候，我们才会真正地创建模块。
func IsolatedChat(factory Factory, shouldRegister conds.Predicate) Module {
	return &isolatedChatModule{
		registeredMods: make(map[int64]Module),
		factory:        factory,
		shouldRegister: shouldRegister,
	}
}
