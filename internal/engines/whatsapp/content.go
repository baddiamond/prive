package whatsapp

import (
	"go.mau.fi/whatsmeow/proto/waE2E"

	"github.com/baddiamond/prive/internal/core"
)

// content returns what a message shows in Privé: its kind and its text (body or caption).
// ok is false for messages that are not shown on their own: reactions, edits, deletions,
// poll votes and other protocol traffic.
func content(m *waE2E.Message) (kind core.MessageKind, text string, ok bool) {
	if m == nil {
		return "", "", false
	}
	switch {
	case m.Conversation != nil:
		return core.MessageText, m.GetConversation(), true
	case m.ExtendedTextMessage != nil:
		return core.MessageText, m.GetExtendedTextMessage().GetText(), true
	case m.ImageMessage != nil:
		return core.MessageImage, m.GetImageMessage().GetCaption(), true
	case m.VideoMessage != nil:
		return core.MessageVideo, m.GetVideoMessage().GetCaption(), true
	case m.PtvMessage != nil:
		return core.MessageVideo, m.GetPtvMessage().GetCaption(), true
	case m.AudioMessage != nil:
		if m.GetAudioMessage().GetPTT() {
			return core.MessageVoice, "", true
		}
		return core.MessageAudio, "", true
	case m.DocumentMessage != nil:
		doc := m.GetDocumentMessage()
		if caption := doc.GetCaption(); caption != "" {
			return core.MessageDocument, caption, true
		}
		return core.MessageDocument, doc.GetFileName(), true
	case m.DocumentWithCaptionMessage != nil:
		return content(m.GetDocumentWithCaptionMessage().GetMessage())
	case m.StickerMessage != nil:
		return core.MessageSticker, "", true
	case m.LocationMessage != nil:
		return core.MessageLocation, m.GetLocationMessage().GetName(), true
	case m.LiveLocationMessage != nil:
		return core.MessageLocation, m.GetLiveLocationMessage().GetCaption(), true
	case m.ContactMessage != nil:
		return core.MessageContact, m.GetContactMessage().GetDisplayName(), true
	case m.ContactsArrayMessage != nil:
		return core.MessageContact, m.GetContactsArrayMessage().GetDisplayName(), true
	case m.PollCreationMessage != nil:
		return core.MessagePoll, m.GetPollCreationMessage().GetName(), true
	case m.PollCreationMessageV2 != nil:
		return core.MessagePoll, m.GetPollCreationMessageV2().GetName(), true
	case m.PollCreationMessageV3 != nil:
		return core.MessagePoll, m.GetPollCreationMessageV3().GetName(), true
	// Anything with visible content was matched above, so what is left here is protocol
	// traffic (a bare SenderKeyDistributionMessage only carries group keys).
	case m.ProtocolMessage != nil, m.ReactionMessage != nil, m.EncReactionMessage != nil,
		m.PollUpdateMessage != nil, m.KeepInChatMessage != nil, m.SenderKeyDistributionMessage != nil,
		m.PinInChatMessage != nil, m.EditedMessage != nil:
		return "", "", false
	}
	return core.MessageUnsupported, "", true
}
