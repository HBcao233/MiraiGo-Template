package ping

import (
	"regexp"

	"github.com/Logiase/MiraiGo-Template/client"
	"github.com/Logiase/MiraiGo-Template/global/coolq"
	Message "github.com/Mrs4s/MiraiGo/message"
	log "github.com/sirupsen/logrus"
)

var pattern = regexp.MustCompile(`(?m)^(\[CQ:.*?\] *)*ping!?$`)

func ping(bot *coolq.CQBot, event *coolq.Event) {
	if event.Raw.PostType != "message" && event.Raw.PostType != "message_sent" {
		return
	}
	if event.Raw.DetailType != "group" {
		return
	}
	log.Infof("event: %s", event.JSONString())

	message, _ := event.Raw.Others["raw_message"].(string)
	match := pattern.FindStringSubmatch(message)
	if len(match) > 0 {
		group_id, _ := event.Raw.Others["group_id"].(int64)
		m := new(Message.SendingMessage)
		m.Elements = make([]Message.IMessageElement, 1)
		m.Elements = append(m.Elements, Message.NewText("pong!"))
		bot.SendGroupMessage(group_id, m)
	}
}

func init() {
	client.AddHandler(ping)
}
