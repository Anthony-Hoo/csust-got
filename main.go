package main

import (
	"csust-got/base"
	"csust-got/config"
	"csust-got/context"
	"csust-got/manage"
	"csust-got/module"
	"csust-got/module/preds"
	"csust-got/orm"
	"csust-got/timer"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(config.BotConfig.Token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = config.BotConfig.DebugMode

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	ctx := context.Global(orm.GetClient(), config.BotConfig)
	handles := module.Parallel([]module.Module{
		module.Stateless(base.Hello, preds.IsCommand("say_hello")),
		module.Stateless(base.WelcomeNewMember, preds.NonEmpty),
		module.Stateless(base.HelloToAll, preds.IsCommand("hello_to_all")),
		module.SharedContext([]module.Module{
			module.WithPredicate(module.IsolatedChat(manage.NoSticker), preds.IsCommand("no_sticker")),
			module.WithPredicate(module.IsolatedChat(manage.DeleteSticker), preds.HasSticker)}),
		module.Stateless(manage.BanMyself, preds.IsCommand("ban_myself")),
		module.Stateless(base.FakeBanMyself, preds.IsCommand("fake_ban_myself")),
		module.Stateless(manage.Ban, preds.IsCommand("ban")),
		module.Stateless(manage.SoftBan, preds.IsCommand("ban_soft")),
		timer.RunTask(),
	})
	handles = module.Sequential([]module.Module{
		module.IsolatedChat(base.Shutdown),
		module.IsolatedChat(manage.FakeBan),
		handles,
	})
	for update := range updates {
		go handles.HandleUpdate(ctx, update, bot)
	}
}
