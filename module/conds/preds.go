package conds

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

// Predicate is the common interface of a 'condition' that indices whether an update should be handled.
type Predicate struct {
	predicate
}

func (p Predicate) And(other Predicate) Predicate {
	return Predicate{andPredicate{p, other}}
}

// SideEffect will be triggered if the predicate is true.
func (p Predicate) SideEffectOnTrue(sideEffect func(update tgbotapi.Update)) Predicate {
	next := func(update tgbotapi.Update) bool {
		sideEffect(update)
		return true
	}
	return p.And(BoolFunction(next))
}

type predicate interface {
	Test(update tgbotapi.Update) bool
}

type andPredicate struct {
	lhs predicate
	rhs predicate
}

func (p andPredicate) Test(update tgbotapi.Update) bool {
	return p.lhs.Test(update) && p.rhs.Test(update)
}

type functionalPredicate struct {
	pred func(update tgbotapi.Update) bool
}

func (f functionalPredicate) Test(update tgbotapi.Update) bool {
	return f.pred(update)
}

func BoolFunction(pred func(update tgbotapi.Update) bool) Predicate {
	return Predicate{functionalPredicate{pred: pred}}
}

// NonEmpty is the condition of a module which only processes non-empty message.
var NonEmpty Predicate = BoolFunction(nonEmpty)

// IsAnyCommand is the condition of a module which only process Command message.
var IsAnyCommand Predicate = BoolFunction(command)

// IsCommand handles the update when the command is exactly the argument.
func IsCommand(command string) Predicate {
	isThat := func(update tgbotapi.Update) bool {
		return update.Message.Command() == command
	}
	return NonEmpty.And(IsAnyCommand).And(BoolFunction(isThat))
}

func nonEmpty(update tgbotapi.Update) bool {
	return update.Message != nil
}

func command(update tgbotapi.Update) bool {
	return nonEmpty(update) && update.Message.IsCommand()
}
