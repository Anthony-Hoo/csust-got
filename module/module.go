package module

import (
	"csust-got/context"
	"csust-got/module/preds"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type HandleResult int

const (
	NextOfChain HandleResult = iota
	NoMore
)

type Module interface {
	// HandleUpdate should handle a update, and return whether the next handler of chain should be handed.
	// Note your module might be executed 'parallel' default, which will ignore your returning value.
	// If you want to register a 'chain of filter', use `Chain` please.
	HandleUpdate(context context.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI) HandleResult
}
type HandleFunc func(update tgbotapi.Update, bot *tgbotapi.BotAPI)
type StatefulHandleFunc func(ctx context.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI)
type trivialModule struct {
	handleUpdate StatefulHandleFunc
}

func (t trivialModule) HandleUpdate(ctx context.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI) HandleResult {
	t.handleUpdate(ctx, update, bot)
	return NextOfChain
}

// Stateful warps a stateful function to a Module.
func Stateful(f StatefulHandleFunc) Module {
	return trivialModule{handleUpdate: f}
}

// WithPredicate warps a Module with specified Predicate.
// The method `handleUpdate` will only be invoked when the Predicate is true.
func WithPredicate(m Module, p preds.Predicate) Module {
	return Stateful(func(ctx context.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		if p.ShouldHandle(update) {
			m.HandleUpdate(ctx, update, bot)
		}
	})
}

// Stateless creates a 'stateless' module.
// If your state is tiny(which can be captured in a closure), use this.
func Stateless(handleFunc HandleFunc, condFunc preds.Predicate) Module {
	handler := Stateful(func(_ context.Context, update tgbotapi.Update, bot *tgbotapi.BotAPI) {
		handleFunc(update, bot)
	})
	return WithPredicate(handler, condFunc)
}
