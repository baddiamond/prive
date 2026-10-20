package whatsapp

import (
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"google.golang.org/protobuf/proto"
)

func markRead() *waSyncAction.MarkChatAsReadAction {
	return &waSyncAction.MarkChatAsReadAction{Read: proto.Bool(true)}
}
