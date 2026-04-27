package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func (s *Session) connectWhatsmeow() {
	client := s.client

	if client.Store.ID == nil {
		// No session yet — need QR auth
		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			bridgeLog.Errorf("[%s] GetQRChannel error: %v", s.name, err)
			s.manager.sendEvent(s.name, "disconnected", map[string]interface{}{
				"reason": fmt.Sprintf("QR channel failed: %v", err),
			})
			return
		}
		err = client.Connect()
		if err != nil {
			bridgeLog.Errorf("[%s] Connect error: %v", s.name, err)
			s.manager.sendEvent(s.name, "disconnected", map[string]interface{}{
				"reason": fmt.Sprintf("connect failed: %v", err),
			})
			return
		}

		// Forward QR codes to TS
		go func() {
			for evt := range qrChan {
				switch evt.Event {
				case "code":
					s.manager.sendEvent(s.name, "qr", map[string]interface{}{
						"code": evt.Code,
					})
				case "success":
					// Connected event will be sent by the event handler
				case "timeout":
					s.manager.sendEvent(s.name, "disconnected", map[string]interface{}{
						"reason": "QR code timeout",
					})
				}
			}
		}()
	} else {
		// Already have session, just connect
		err := client.Connect()
		if err != nil {
			bridgeLog.Errorf("[%s] Connect error: %v", s.name, err)
			s.manager.sendEvent(s.name, "disconnected", map[string]interface{}{
				"reason": fmt.Sprintf("connect failed: %v", err),
			})
		}
	}
}

func (s *Session) handleWhatsmeowEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		jid := ""
		if s.client.Store.ID != nil {
			jid = s.client.Store.ID.String()
			// Persist name → JID mapping so we can find this device on restart
			s.manager.setNameMapping(s.name, jid)
		}
		if autoPresence {
			if err := s.client.SendPresence(context.Background(), types.PresenceAvailable); err != nil {
				bridgeLog.Warnf("[%s] Failed to send presence: %v", s.name, err)
			}
		}
		s.manager.sendEvent(s.name, "connected", map[string]interface{}{
			"jid": jid,
		})

	case *events.Disconnected:
		s.manager.sendEvent(s.name, "disconnected", map[string]interface{}{
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

		s.manager.sendEvent(s.name, "receipt", map[string]interface{}{
			"from":       v.MessageSource.Sender.String(),
			"chat":       v.MessageSource.Chat.String(),
			"messageIds": v.MessageIDs,
			"type":       receiptType,
			"timestamp":  v.Timestamp.Unix(),
		})

	case *events.Presence:
		presenceType := "unavailable"
		if !v.Unavailable {
			presenceType = "available"
		}
		s.manager.sendEvent(s.name, "presence", map[string]interface{}{
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
		s.manager.sendEvent(s.name, "group_join", map[string]interface{}{
			"chat":         v.JID.String(),
			"participants": participants,
		})

	case *events.GroupInfo:
		if v.Name != nil {
			s.manager.sendEvent(s.name, "group_update", map[string]interface{}{
				"chat":      v.JID.String(),
				"field":     "name",
				"value":     v.Name.Name,
				"updatedBy": v.Sender.String(),
			})
		}
		if v.Topic != nil {
			s.manager.sendEvent(s.name, "group_update", map[string]interface{}{
				"chat":      v.JID.String(),
				"field":     "topic",
				"value":     v.Topic.Topic,
				"updatedBy": v.Sender.String(),
			})
		}
	}
}

func extractQuotedInfo(ci *waE2E.ContextInfo, fallbackSender string) map[string]interface{} {
	if ci == nil || ci.StanzaID == nil {
		return nil
	}

	sender := ci.GetParticipant()
	if sender == "" {
		sender = fallbackSender
	}

	quoted := map[string]interface{}{
		"messageId": ci.GetStanzaID(),
		"sender":    sender,
	}

	if qm := ci.GetQuotedMessage(); qm != nil {
		switch {
		case qm.GetConversation() != "":
			quoted["type"] = "text"
			quoted["text"] = qm.GetConversation()
		case qm.GetExtendedTextMessage() != nil:
			quoted["type"] = "text"
			quoted["text"] = qm.GetExtendedTextMessage().GetText()
		case qm.GetImageMessage() != nil:
			quoted["type"] = "image"
			quoted["caption"] = qm.GetImageMessage().GetCaption()
			quoted["mimetype"] = qm.GetImageMessage().GetMimetype()
		case qm.GetVideoMessage() != nil:
			quoted["type"] = "video"
			quoted["caption"] = qm.GetVideoMessage().GetCaption()
			quoted["mimetype"] = qm.GetVideoMessage().GetMimetype()
		case qm.GetAudioMessage() != nil:
			quoted["type"] = "audio"
			quoted["mimetype"] = qm.GetAudioMessage().GetMimetype()
		case qm.GetDocumentMessage() != nil:
			quoted["type"] = "document"
			quoted["caption"] = qm.GetDocumentMessage().GetCaption()
			quoted["mimetype"] = qm.GetDocumentMessage().GetMimetype()
			quoted["filename"] = qm.GetDocumentMessage().GetFileName()
		case qm.GetStickerMessage() != nil:
			quoted["type"] = "sticker"
			quoted["mimetype"] = qm.GetStickerMessage().GetMimetype()
		case qm.GetContactMessage() != nil:
			quoted["type"] = "contact"
			quoted["displayName"] = qm.GetContactMessage().GetDisplayName()
		case qm.GetLocationMessage() != nil:
			quoted["type"] = "location"
			quoted["latitude"] = qm.GetLocationMessage().GetDegreesLatitude()
			quoted["longitude"] = qm.GetLocationMessage().GetDegreesLongitude()
		}
	}

	return quoted
}

func (s *Session) handleMessageEvent(v *events.Message) {
	if v.Info.IsFromMe && !subscribeOutgoing {
		return
	}

	msg := v.Message
	isViewOnce := v.IsViewOnce

	base := map[string]interface{}{
		"id":        v.Info.ID,
		"from":      v.Info.Sender.String(),
		"chat":      v.Info.Chat.String(),
		"pushName":  v.Info.PushName,
		"timestamp": v.Info.Timestamp.Unix(),
		"isGroup":   v.Info.IsGroup,
		"isFromMe":  v.Info.IsFromMe,
	}

	// Reactions use a separate event type
	if rm := msg.GetReactionMessage(); rm != nil {
		s.manager.sendEvent(s.name, "message_reaction", map[string]interface{}{
			"from":      v.Info.Sender.String(),
			"chat":      v.Info.Chat.String(),
			"messageId": rm.GetKey().GetID(),
			"emoji":     rm.GetText(),
			"timestamp": v.Info.Timestamp.Unix(),
		})
		return
	}

	// Helper to attach quoted message info from ContextInfo
	attachQuoted := func(ci *waE2E.ContextInfo) {
		if q := extractQuotedInfo(ci, v.Info.Chat.String()); q != nil {
			base["quotedMessage"] = q
		}
	}

	switch {
	case msg.GetConversation() != "":
		base["type"] = "text"
		base["text"] = msg.GetConversation()

	case msg.GetExtendedTextMessage() != nil:
		base["type"] = "text"
		base["text"] = msg.GetExtendedTextMessage().GetText()
		attachQuoted(msg.GetExtendedTextMessage().GetContextInfo())

	case msg.GetImageMessage() != nil:
		im := msg.GetImageMessage()
		base["type"] = "image"
		base["caption"] = im.GetCaption()
		base["mimetype"] = im.GetMimetype()
		base["mediaRef"] = encodeMediaRef("image", im)
		if isViewOnce || im.GetViewOnce() {
			base["viewOnce"] = true
		}
		if im.GetWidth() > 0 {
			base["width"] = im.GetWidth()
		}
		if im.GetHeight() > 0 {
			base["height"] = im.GetHeight()
		}
		attachQuoted(im.GetContextInfo())

	case msg.GetVideoMessage() != nil:
		vm := msg.GetVideoMessage()
		base["type"] = "video"
		base["caption"] = vm.GetCaption()
		base["mimetype"] = vm.GetMimetype()
		base["mediaRef"] = encodeMediaRef("video", vm)
		if isViewOnce || vm.GetViewOnce() {
			base["viewOnce"] = true
		}
		if vm.GetWidth() > 0 {
			base["width"] = vm.GetWidth()
		}
		if vm.GetHeight() > 0 {
			base["height"] = vm.GetHeight()
		}
		attachQuoted(vm.GetContextInfo())

	case msg.GetAudioMessage() != nil:
		am := msg.GetAudioMessage()
		base["type"] = "audio"
		base["mimetype"] = am.GetMimetype()
		base["mediaRef"] = encodeMediaRef("audio", am)
		attachQuoted(am.GetContextInfo())

	case msg.GetDocumentMessage() != nil:
		dm := msg.GetDocumentMessage()
		base["type"] = "document"
		base["caption"] = dm.GetCaption()
		base["mimetype"] = dm.GetMimetype()
		base["filename"] = dm.GetFileName()
		base["mediaRef"] = encodeMediaRef("document", dm)
		attachQuoted(dm.GetContextInfo())

	case msg.GetStickerMessage() != nil:
		sm := msg.GetStickerMessage()
		base["type"] = "sticker"
		base["mimetype"] = sm.GetMimetype()
		base["mediaRef"] = encodeMediaRef("sticker", sm)
		attachQuoted(sm.GetContextInfo())

	case msg.GetContactMessage() != nil:
		cm := msg.GetContactMessage()
		base["type"] = "contact"
		base["displayName"] = cm.GetDisplayName()
		base["vcard"] = cm.GetVcard()
		attachQuoted(cm.GetContextInfo())

	case msg.GetLocationMessage() != nil:
		lm := msg.GetLocationMessage()
		base["type"] = "location"
		base["latitude"] = lm.GetDegreesLatitude()
		base["longitude"] = lm.GetDegreesLongitude()
		base["name"] = lm.GetName()
		base["address"] = lm.GetAddress()
		attachQuoted(lm.GetContextInfo())

	default:
		return
	}

	s.manager.sendEvent(s.name, "message", base)
}

func encodeMediaRef(mediaType string, msg proto.Message) string {
	data, err := proto.Marshal(msg)
	if err != nil {
		bridgeLog.Warnf("Failed to marshal media ref: %v", err)
		return ""
	}
	return fmt.Sprintf("%s:%s", mediaType, base64.StdEncoding.EncodeToString(data))
}