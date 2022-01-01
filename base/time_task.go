package base

import (
	"fmt"
	"time"

	"csust-got/entities"
	"csust-got/util"

	"go.uber.org/zap"
	. "gopkg.in/tucnak/telebot.v3"
)

// RunTask can run a task.
func RunTask(ctx Context) error {
	text := "你嗦啥，我听不太懂欸……"

	cmd := entities.FromMessage(ctx.Message())
	delay, err := util.EvalDuration(cmd.Arg(0))
	info := cmd.ArgAllInOneFrom(1)
	if err != nil || delay < time.Second {
		return ctx.Reply(text)
	}

	text = fmt.Sprintf("好的，在 %v 后我会来叫你……`%s`，嗯，不愧是我。", delay, info)
	task := func() {
		uid := ctx.Sender().Username
		hint := fmt.Sprintf("@%s 我来了，你要我提醒你……`%s`，大概没错吧。", uid, info)
		err := ctx.Send(hint, ModeMarkdownV2)
		if err != nil {
			zap.L().Error("Run Task send msg to user failed",
				zap.String("user", uid),
				zap.String("msg", info),
				zap.Error(err),
			)
		}
	}
	time.AfterFunc(delay, task)
	return ctx.Reply(text)
}
