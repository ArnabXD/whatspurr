package main

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (s *Session) connectWhatsmeow() {
	client := s.client.(*whatsmeow.Client)

	if client.Store.ID == nil {
		// No session yet — need QR auth
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			bridgeLog.Errorf("Connect error: %v", err)
			s.sendEvent("disconnected", map[string]interface{}{
				"reason": fmt.Sprintf("connect failed: %v", err),
			})
			return
		}

		// Forward QR codes to TS
		go func() {
			for evt := range qrChan {
				switch evt.Event {
				case "code":
					s.sendEvent("qr", map[string]interface{}{
						"code": evt.Code,
					})
				case "success":
					// Connected event will be sent by the event handler
				case "timeout":
					s.sendEvent("disconnected", map[string]interface{}{
						"reason": "QR code timeout",
					})
				}
			}
		}()
	} else {
		// Already have session, just connect
		err := client.Connect()
		if err != nil {
			bridgeLog.Errorf("Connect error: %v", err)
			s.sendEvent("disconnected", map[string]interface{}{
				"reason": fmt.Sprintf("connect failed: %v", err),
			})
		}
	}
}

func (s *Session) handleWhatsmeowEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		client := s.client.(*whatsmeow.Client)
		jid := ""
		if client.Store.ID != nil {
			jid = client.Store.ID.String()
		}
		// Auto-send presence so we receive read receipts and presence updates
		if autoPresence {
			if err := client.SendPresence(context.Background(), types.PresenceAvailable); err != nil {
				bridgeLog.Warnf("Failed to send presence: %v", err)
			}
		}
		s.sendEvent("connected", map[string]interface{}{
			"jid": jid,
		})

	case *events.Disconnected:
		s.sendEvent("disconnected", map[string]interface{}{
			"reason": "disconnected",
		})

	case *events.Message:
		s.handleMessageEvent(v)

	case *events.Receipt:
		receiptType := "delivered"
		switch v.Type {
		case types.ReceiptTypeRead:
			receiptType = "read"
		case types.ReceiptTypeDelivered:
			receiptType = "delivered"
		}

		msgIds := make([]string, len(v.MessageIDs))
		copy(msgIds, v.MessageIDs)

		s.sendEvent("receipt", map[string]interface{}{
			"from":       v.MessageSource.Sender.String(),
			"chat":       v.MessageSource.Chat.String(),
			"messageIds": msgIds,
			"type":       receiptType,
			"timestamp":  v.Timestamp.Unix(),
		})

	case *events.Presence:
		presenceType := "unavailable"
		if v.Unavailable == false {
			presenceType = "available"
		}
		s.sendEvent("presence", map[string]interface{}{
			"from":     v.From.String(),
			"chat":     v.From.String(),
			"type":     presenceType,
			"lastSeen": v.LastSeen.Unix(),
		})

	case *events.JoinedGroup:
		participants := make([]string, len(v.GroupInfo.Participants))
		for i, p := range v.GroupInfo.Participants {
			participants[i] = p.JID.String()
		}
		s.sendEvent("group_join", map[string]interface{}{
			"chat":         v.JID.String(),
			"participants": participants,
		})

	case *events.GroupInfo:
		if v.Name != nil {
			s.sendEvent("group_update", map[string]interface{}{
				"chat":      v.JID.String(),
				"field":     "name",
				"value":     v.Name.Name,
				"updatedBy": v.Sender.String(),
			})
		}
		if v.Topic != nil {
			s.sendEvent("group_update", map[string]interface{}{
				"chat":      v.JID.String(),
				"field":     "topic",
				"value":     v.Topic.Topic,
				"updatedBy": v.Sender.String(),
			})
		}
	}
}

func (s *Session) handleMessageEvent(v *events.Message) {
	// Skip outgoing messages unless subscribed
	if v.Info.IsFromMe && !subscribeOutgoing {
		return
	}

	msg := v.Message

	base := map[string]interface{}{
		"id":        v.Info.ID,
		"from":      v.Info.Sender.String(),
		"chat":      v.Info.Chat.String(),
		"pushName":  v.Info.PushName,
		"timestamp": v.Info.Timestamp.Unix(),
		"isGroup":   v.Info.IsGroup,
		"isFromMe":  v.Info.IsFromMe,
	}

	switch {
	case msg.GetConversation() != "":
		base["type"] = "text"
		base["text"] = msg.GetConversation()
		s.sendEvent("message", base)

	case msg.GetExtendedTextMessage() != nil:
		base["type"] = "text"
		base["text"] = msg.GetExtendedTextMessage().GetText()
		s.sendEvent("message", base)

	case msg.GetImageMessage() != nil:
		im := msg.GetImageMessage()
		base["type"] = "image"
		base["caption"] = im.GetCaption()
		base["mimetype"] = im.GetMimetype()
		base["mediaRef"] = encodeMediaRef(v)
		s.sendEvent("message", base)

	case msg.GetVideoMessage() != nil:
		vm := msg.GetVideoMessage()
		base["type"] = "video"
		base["caption"] = vm.GetCaption()
		base["mimetype"] = vm.GetMimetype()
		base["mediaRef"] = encodeMediaRef(v)
		s.sendEvent("message", base)

	case msg.GetAudioMessage() != nil:
		am := msg.GetAudioMessage()
		base["type"] = "audio"
		base["mimetype"] = am.GetMimetype()
		base["mediaRef"] = encodeMediaRef(v)
		s.sendEvent("message", base)

	case msg.GetDocumentMessage() != nil:
		dm := msg.GetDocumentMessage()
		base["type"] = "document"
		base["caption"] = dm.GetCaption()
		base["mimetype"] = dm.GetMimetype()
		base["filename"] = dm.GetFileName()
		base["mediaRef"] = encodeMediaRef(v)
		s.sendEvent("message", base)

	case msg.GetStickerMessage() != nil:
		sm := msg.GetStickerMessage()
		base["type"] = "sticker"
		base["mimetype"] = sm.GetMimetype()
		base["mediaRef"] = encodeMediaRef(v)
		s.sendEvent("message", base)

	case msg.GetContactMessage() != nil:
		cm := msg.GetContactMessage()
		base["type"] = "contact"
		base["displayName"] = cm.GetDisplayName()
		base["vcard"] = cm.GetVcard()
		s.sendEvent("message", base)

	case msg.GetLocationMessage() != nil:
		lm := msg.GetLocationMessage()
		base["type"] = "location"
		base["latitude"] = lm.GetDegreesLatitude()
		base["longitude"] = lm.GetDegreesLongitude()
		base["name"] = lm.GetName()
		base["address"] = lm.GetAddress()
		s.sendEvent("message", base)

	case msg.GetReactionMessage() != nil:
		rm := msg.GetReactionMessage()
		s.sendEvent("message_reaction", map[string]interface{}{
			"from":      v.Info.Sender.String(),
			"chat":      v.Info.Chat.String(),
			"messageId": rm.GetKey().GetID(),
			"emoji":     rm.GetText(),
			"timestamp": v.Info.Timestamp.Unix(),
		})
	}
}

// encodeMediaRef creates an opaque reference string for media download.
// For now we store the message ID — the download_media command will
// look it up from whatsmeow's message store.
func encodeMediaRef(v *events.Message) string {
	// Store enough info to reconstruct the download later.
	// We JSON-encode the essential fields.
	return fmt.Sprintf("%s/%s/%s", v.Info.Chat.String(), v.Info.Sender.String(), v.Info.ID)
}