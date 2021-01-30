package prom

import (
	"csust-got/entities"
	"go.uber.org/zap"
	. "gopkg.in/tucnak/telebot.v2"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var host, _ = os.Hostname()

func InitPrometheus() {
	prometheus.MustRegister(commandTimes)
	prometheus.MustRegister(messageCount)
	prometheus.MustRegister(updateCostTime)
	prometheus.MustRegister(chatMemberCount)
	prometheus.MustRegister(newMemberCount)
	prometheus.MustRegister(logCount)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			zap.L().Error("InitPrometheus: Serve http failed", zap.Error(err))
			Log(zap.ErrorLevel.String())
		}
	}()
}

func newLabels(base, labels prometheus.Labels) prometheus.Labels {
	for k, v := range base {
		labels[k] = v
	}
	return labels
}

// DailUpdate - dail an update
func DailUpdate(m *Message, valied bool, costTime time.Duration) {
	if !valied || m.Private() {
		return
	}
	labels := prometheus.Labels{"host": host}

	chat := m.Chat
	labels["chat_name"] = chat.Title

	user := m.Sender
	if user == nil || user.IsBot {
		return
	}
	username := user.Username
	if username == "" {
		username = user.FirstName
	}
	labels["username"] = username

	isCommand, isSticker := "false", "false"

	if m.Sticker != nil {
		isSticker = "true"
	}

	command := entities.FromMessage(m)
	if command != nil {
		isCommand = "true"
		commandTimes.With(newLabels(labels, prometheus.Labels{
			"command_name": command.Name(),
		})).Inc()
	}

	updateCostTime.With(labels).Set(float64(costTime.Nanoseconds()) / 1e6)

	messageCount.With(newLabels(labels, prometheus.Labels{
		"is_command": isCommand,
		"is_sticker": isSticker,
	})).Inc()
}

func NewMember(chatName string) {
	newMemberCount.With(prometheus.Labels{
		"host":      host,
		"chat_name": chatName,
	}).Inc()
}

func MemberLeft(chatName string) {
	chatMemberCount.With(prometheus.Labels{
		"host":      host,
		"chat_name": chatName,
	}).Desc()
}

func GetMember(chatName string, num int) {
	chatMemberCount.With(prometheus.Labels{
		"host":      host,
		"chat_name": chatName,
	}).Set(float64(num))
}

func Log(level string) {
	logCount.With(prometheus.Labels{
		"host":  host,
		"level": level,
	}).Inc()
}
