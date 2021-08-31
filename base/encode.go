package base

import (
	"csust-got/entities"
	"strings"

	. "gopkg.in/tucnak/telebot.v3"
)

// change 'y' to 'i' if end with this
var xyTable = [...]string{"ty", "ly", "fy", "py", "dy", "by"}

// HugeEncoder encode 'xxx' to 'hugexxxer'
func HugeEncoder(ctx Context) error {
	command := entities.FromMessage(ctx.Message())

	// no args
	if command.Argc() <= 0 {
		return ctx.Reply("HUGEFIVER")
	}

	args := command.MultiArgsFrom(0)

	// tldr
	if len(args) > 10 {
		return ctx.Reply("hugeTLDRer")
	}

	for i := range args {
		// tldr
		if len(args[i]) > 20 {
			args[i] = "hugeTLDRer"
			continue
		}
		// add 'huge' to prefix
		if !strings.HasPrefix(args[i], "huge") {
			if args[i][0] == 'e' {
				args[i] = "hug" + args[i]
			} else {
				args[i] = "huge" + args[i]
			}
		}
		// add 'er' to suffix
		if !strings.HasSuffix(args[i], "er") {
			// change 'y' to 'i' if end with $xyTable
			for _, v := range xyTable {
				if strings.HasSuffix(args[i], v) {
					args[i] = args[i][0:len(args[i])-1] + "i"
					break
				}
			}
			// only add 'r' if $args[i] end with 'e'
			if args[i][len(args[i])-1] == 'e' {
				args[i] = args[i] + "r"
			} else {
				args[i] = args[i] + "er"
			}
		}
		// if we get 'huger' after encode, we <fork> him.
		if args[i] == "huger" {
			args[i] = "hugeF**Ker"
		}
	}

	return ctx.Reply(strings.Join(args, "\n"))
}

// HugeDecoder decode 'hugehugehugexxxererer' to 'hugexxxer'
func HugeDecoder(ctx Context) error {
	command := entities.FromMessage(ctx.Message())

	// no args
	if command.Argc() <= 0 {
		return ctx.Reply("HUGEFIVER")
	}

	arg := command.ArgAllInOneFrom(0)

	// tldr
	if len(arg) > 500 {
		return ctx.Reply("hugeTLDRer")
	}

	// find first 'huge' and last 'er'
	huge := strings.Index(arg, "huge")
	er := strings.LastIndex(arg, "er")

	// can't find any 'huge' or 'er'
	if huge == -1 || er == -1 {
		return ctx.Reply("hugeNOTFOUNDer")
	}

	// if first 'huge' after last 'er'
	if huge > er {
		return ctx.Reply("hugeFAKEr")
	}

	// find end of first consecutive 'huge' and start of last consecutive 'er'
	var hugeEnd, erStart int
	for hugeEnd = huge; hugeEnd+4 < len(arg); hugeEnd += 4 {
		if arg[hugeEnd:hugeEnd+4] != "huge" {
			break
		}
	}
	for erStart = er; erStart-2 >= 0; erStart -= 2 {
		if arg[erStart-2:erStart] != "er" {
			break
		}
	}

	// if we will get 'huger', we <fork> him.
	if erStart < hugeEnd {
		return ctx.Reply("hugeF**Ker")
	}

	// decode
	arg = arg[0:huge+4] + arg[hugeEnd:erStart] + arg[er:]

	return ctx.Reply(arg)
}
