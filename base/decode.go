package base

import (
	"csust-got/entities"
	"csust-got/util"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf16"

	exencoding "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	tb "gopkg.in/telebot.v3"
)

// DecodeCommandPatt is regex patt of this command
var DecodeCommandPatt = regexp.MustCompile(`^decode(?:_([a-zA-Z\d\.\-]+)?(?:_([a-zA-Z\d\.\-]+))?)?(?:@.*)?\s*$`)

var errInvalidCmd = errors.New("invalid command")

// Decode decode text command
// nolint:goconst
func Decode(ctx tb.Context) error {
	cmd, text, err := entities.CommandTakeArgs(ctx.Message(), 0)
	if err != nil {
		return err
	}

	grps := DecodeCommandPatt.FindAllStringSubmatch(cmd.Name(), -1)
	if len(grps) == 0 {
		return errInvalidCmd
	}

	from, to := "utf8", "utf8"
	if len(grps[0]) >= 2 {
		from = normalizeEncoding(grps[0][1])
		if from == "" {
			from = "utf8"
		}
	}
	if len(grps[0]) >= 3 {
		to = normalizeEncoding(grps[0][2])
		if to == "" {
			to = "utf8"
		}
	}

	var bs []byte
	var encoder *exencoding.Encoder
	useEncoder := true

	switch from {
	case "gbk":
		encoder = simplifiedchinese.GBK.NewEncoder()
	case "gb18030":
		encoder = simplifiedchinese.GB18030.NewEncoder()
	case "big5":
		encoder = traditionalchinese.Big5.NewEncoder()
	case "shift-jis":
		encoder = japanese.ShiftJIS.NewEncoder()
	case "windows-1252":
		encoder = charmap.Windows1252.NewEncoder()
	default:
		useEncoder = false
	}

	if useEncoder {
		bs, _ = encoder.Bytes([]byte(text))
	} else {
		switch from {
		case "base64":
			bs, err = base64.StdEncoding.DecodeString(text)
			if err != nil {
				return err
			}
		case "hex":
			bs, err = hex.DecodeString(text)
			if err != nil {
				return err
			}
		case "utf8":
			bs = []byte(text)
		case "utf16le":
			s := utf16.Encode([]rune(text))
			bs = make([]byte, len(s)*2)
			for i, v := range s {
				bs[i*2] = byte(v)
				bs[i*2+1] = byte(v >> 8)
			}
		case "utf16be":
			s := utf16.Encode([]rune(text))
			bs = make([]byte, len(s)*2)
			for i, v := range s {
				bs[i*2] = byte(v >> 8)
				bs[i*2+1] = byte(v)
			}
		}
	}

	var result string
	var decoder *exencoding.Decoder
	useDecoder := true

	switch to {
	case "gbk":
		decoder = simplifiedchinese.GBK.NewDecoder()
	case "gb18030":
		decoder = simplifiedchinese.GB18030.NewDecoder()
	case "big5":
		decoder = traditionalchinese.Big5.NewDecoder()
	case "shift-jis":
		decoder = japanese.ShiftJIS.NewDecoder()
	case "windows-1252":
		decoder = charmap.Windows1252.NewDecoder()
	default:
		useDecoder = false
	}

	if useDecoder {
		bs, _ = decoder.Bytes(bs)
		result = string(bs)
	} else {
		switch to {
		case "base64":
			result = base64.StdEncoding.EncodeToString(bs)
		case "hex":
			result = hex.EncodeToString(bs)
		case "utf8":
			result = string(bs)
		case "utf16le":
			bsLen := len(bs)
			if bsLen%2 != 0 {
				bsLen--
			}
			s := make([]uint16, bsLen/2)
			for i := 0; i < len(s); i++ {
				s[i] = uint16(bs[i*2]) | uint16(bs[i*2+1])<<8
			}
			result = string(utf16.Decode(s))
		case "utf16be":
			bsLen := len(bs)
			if bsLen%2 != 0 {
				bsLen--
			}
			s := make([]uint16, bsLen/2)
			for i := 0; i < len(s); i++ {
				s[i] = uint16(bs[i*2+1]) | uint16(bs[i*2])<<8
			}
		}
	}

	result = fmt.Sprintf("```%s```", escapeMdReservedChars(result))

	util.SendReply(ctx.Chat(), result, ctx.Message(), tb.ModeMarkdownV2)

	return nil
}

func normalizeEncoding(in string) string {
	encoding := strings.ToLower(in)
	switch encoding {
	case "utf8", "utf-8":
		return "utf8"
	case "gbk", "gb2312":
		return "gbk"
	case "gb18030":
		return "gb18030"
	case "big5":
		return "big5"
	case "jp", "shift-jis", "shift_jis":
		return "shift-jis"
	case "windows-1252", "windows1252", "codepage-1252", "codepage1252", "cp1252", "latin1", "latin-1":
		return "windows-1252"
	case "utf16", "utf-16", "utf16le", "utf-16le":
		return "utf16le"
	case "utf16be", "utf-16be":
		return "utf16be"
	case "base64":
		return "base64"
	case "hex":
		return "hex"
	default:
		return "utf8"
	}
}

func escapeMdReservedChars(s string) string {
	reservedChars := []string{"\\", "_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	for _, char := range reservedChars {
		s = strings.ReplaceAll(s, char, "\\"+char)
	}

	return s
}
