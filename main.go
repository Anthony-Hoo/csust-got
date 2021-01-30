package main

import (
	"csust-got/base"
	"csust-got/config"
	"csust-got/entities"
	"csust-got/log"
	"csust-got/orm"
	"csust-got/prom"
	"csust-got/restrict"
	"csust-got/util"
	. "gopkg.in/tucnak/telebot.v2"
	"time"

	"go.uber.org/zap"
)

func main() {
	config.InitConfig("config.yaml", "BOT")
	log.InitLogger()
	defer log.Sync()
	prom.InitPrometheus()
	orm.InitRedis()

	orm.LoadWhiteList()
	orm.LoadBlackList()

	bot, err := initBot()
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	bot.Handle("/say_hello", base.Hello)
	bot.Handle("/hello_to_all", base.HelloToAll)

	bot.Handle("/id", util.PrivateCommand(base.GetUserID))
	bot.Handle("/cid", base.GetChatID)
	bot.Handle("/info", base.Info)
	bot.Handle("/links", base.Links)

	// bot.Handle("/history", base.History)
	bot.Handle("/forward", base.Forward)
	bot.Handle("/mc", base.MC)

	bot.Handle("/sleep", base.Sleep)
	bot.Handle("/no_sleep", base.NoSleep)

	bot.Handle("/google", base.Google)
	bot.Handle("/bing", base.Bing)
	bot.Handle("/bilibili", base.Bilibili)
	bot.Handle("/github", base.Github)

	bot.Handle("/recorder", base.Repeat)

	bot.Handle("/hitokoto", base.Hitokoto)
	bot.Handle("/hitowuta", base.HitDawu)
	bot.Handle("/hitdawu", base.HitDawu)
	bot.Handle("/hito_netease", base.HitoNetease)

	bot.Handle("/run_after", base.RunTask)

	bot.Handle("/fake_ban_myself", base.FakeBanMyself)
	bot.Handle("/fake_ban", util.GroupCommand(restrict.FakeBan))
	bot.Handle("/kill", util.GroupCommand(restrict.FakeBan))
	bot.Handle("/ban_myself", util.GroupCommand(restrict.BanMyself))
	bot.Handle("/ban", util.GroupCommand(restrict.Ban))
	bot.Handle("/ban_soft", util.GroupCommand(restrict.SoftBan))
	bot.Handle("/no_sticker", util.GroupCommand(restrict.NoSticker))
	bot.Handle("/shutdown", util.GroupCommand(base.Shutdown))
	bot.Handle("/halt", util.GroupCommand(base.Shutdown))
	bot.Handle("/boot", util.GroupCommand(base.Boot))

	bot.Handle(OnUserJoined, base.WelcomeNewMember)
	// bot.Handle(OnUserLeft, base.LeftMember)

	bot.Start()
}

func initBot() (*Bot, error) {
	panicReporter := func(err error) {
		log.Error("bot recover form panic", zap.Error(err))
	}

	bot, err := NewBot(Settings{
		Token:     config.BotConfig.Token,
		Updates:   1024,
		ParseMode: ModeMarkdownV2,
		Reporter:  panicReporter,
		Poller:    initPoller(),
	})
	if err != nil {
		return nil, err
	}

	config.BotConfig.Bot = bot
	log.Info("Success Authorized", zap.String("botUserName", bot.Me.Username))
	return bot, nil
}

func initPoller() *MiddlewarePoller {
	defaultPoller := &LongPoller{Timeout: 10 * time.Second}
	blackListPoller := NewMiddlewarePoller(defaultPoller, blackListFilter)
	fakeBanPoller := NewMiddlewarePoller(blackListPoller, fakeBanFilter)
	rateLimitPoller := NewMiddlewarePoller(fakeBanPoller, rateLimitFilter)
	shutdownPoller := NewMiddlewarePoller(rateLimitPoller, shutdownFilter)
	noStickerPoller := NewMiddlewarePoller(shutdownPoller, noStickerFilter)
	return noStickerPoller
}

func blackListFilter(update *Update) bool {
	if update.Message == nil {
		return true
	}
	if config.BotConfig.BlackListConfig.Check(update.Message.Chat.ID) {
		return false
	}
	return true
}

func fakeBanFilter(update *Update) bool {
	if update.Message == nil {
		return true
	}
	m := update.Message
	if orm.IsBanned(m.Chat.ID, m.Sender.ID) {
		return false
	}
	return true
}

func rateLimitFilter(update *Update) bool {
	if update.Message == nil || update.Message.Private() {
		return true
	}
	whiteListConfig := config.BotConfig.WhiteListConfig
	if !whiteListConfig.Enabled || whiteListConfig.Check(update.Message.Chat.ID) {
		return true
	}
	if !restrict.CheckLimit(update.Message) {
		return false
	}
	return true
}

func shutdownFilter(update *Update) bool {
	if update.Message == nil {
		return true
	}
	text := update.Message.Text
	if text == "" {
		return true
	}
	cmd := entities.FromMessage(update.Message)
	if cmd.Name() == "boot" {
		return true
	}
	if orm.IsShutdown(update.Message.Chat.ID) {
		return false
	}
	return true
}

func noStickerFilter(update *Update) bool {
	if update.Message == nil || update.Message.Sticker == nil {
		return true
	}
	if orm.IsNoStickerMode(update.Message.Chat.ID) {
		return false
	}
	return true
}
