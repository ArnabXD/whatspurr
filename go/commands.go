package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func (s *Session) handleCommand(cmd Command) {
	var resp Response
	resp.ID = cmd.ID

	switch cmd.Method {
	case "send_message":
		resp = s.cmdSendMessage(cmd)
	case "send_image":
		resp = s.cmdSendMedia(cmd, "image")
	case "send_video":
		resp = s.cmdSendMedia(cmd, "video")
	case "send_audio":
		resp = s.cmdSendMedia(cmd, "audio")
	case "send_document":
		resp = s.cmdSendMedia(cmd, "document")
	case "send_reaction":
		resp = s.cmdSendReaction(cmd)
	case "get_group_info":
		resp = s.cmdGetGroupInfo(cmd)
	case "set_presence":
		resp = s.cmdSetPresence(cmd)
	default:
		resp.Error = &ErrorInfo{Code: 1002, Message: fmt.Sprintf("unknown method: %s", cmd.Method)}
	}

	resp.ID = cmd.ID
	s.sendResponse(resp)
}

func (s *Session) getClient() *whatsmeow.Client {
	return s.client.(*whatsmeow.Client)
}

func (s *Session) parseJID(params map[string]interface{}, key string) (types.JID, error) {
	raw, ok := params[key].(string)
	if !ok || raw == "" {
		return types.JID{}, fmt.Errorf("missing or invalid '%s' parameter", key)
	}
	jid, err := types.ParseJID(raw)
	if err != nil {
		return types.JID{}, fmt.Errorf("invalid JID '%s': %v", raw, err)
	}
	return jid, nil
}

func (s *Session) cmdSendMessage(cmd Command) Response {
	to, err := s.parseJID(cmd.Params, "to")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	text, _ := cmd.Params["text"].(string)
	if text == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'text' parameter"}}
	}

	resp, err := s.getClient().SendMessage(context.Background(), to, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		bridgeLog.Warnf("send_message error: %v", err)
		return Response{Error: &ErrorInfo{Code: 1004, Message: "send failed"}}
	}

	return Response{Result: map[string]interface{}{
		"messageId": resp.ID,
	}}
}

func (s *Session) cmdSendMedia(cmd Command, mediaType string) Response {
	to, err := s.parseJID(cmd.Params, "to")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	dataB64, _ := cmd.Params["data"].(string)
	if dataB64 == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'data' parameter"}}
	}

	// WhatsApp per-type file size limits
	var maxBytes int
	switch mediaType {
	case "image", "video", "audio":
		maxBytes = 16 * 1024 * 1024 // 16 MB
	case "document":
		maxBytes = 100 * 1024 * 1024 // 100 MB
	default:
		maxBytes = 16 * 1024 * 1024
	}

	if base64.StdEncoding.DecodedLen(len(dataB64)) > maxBytes {
		return Response{Error: &ErrorInfo{Code: 1003, Message: fmt.Sprintf("%s exceeds %d MB limit", mediaType, maxBytes/1024/1024)}}
	}

	data, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "invalid base64 data"}}
	}

	caption, _ := cmd.Params["caption"].(string)
	mimetype, _ := cmd.Params["mimetype"].(string)
	filename, _ := cmd.Params["filename"].(string)

	client := s.getClient()
	var msg *waE2E.Message

	switch mediaType {
	case "image":
		uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			bridgeLog.Warnf("upload error: %v", err)
			return Response{Error: &ErrorInfo{Code: 1005, Message: "media upload failed"}}
		}
		msg = &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(caption),
				Mimetype:      proto.String(mimetype),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
			},
		}
	case "video":
		uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			bridgeLog.Warnf("upload error: %v", err)
			return Response{Error: &ErrorInfo{Code: 1005, Message: "media upload failed"}}
		}
		msg = &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       proto.String(caption),
				Mimetype:      proto.String(mimetype),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
			},
		}
	case "audio":
		uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaAudio)
		if err != nil {
			bridgeLog.Warnf("upload error: %v", err)
			return Response{Error: &ErrorInfo{Code: 1005, Message: "media upload failed"}}
		}
		msg = &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				Mimetype:      proto.String(mimetype),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
			},
		}
	case "document":
		uploaded, err := client.Upload(context.Background(), data, whatsmeow.MediaDocument)
		if err != nil {
			bridgeLog.Warnf("upload error: %v", err)
			return Response{Error: &ErrorInfo{Code: 1005, Message: "media upload failed"}}
		}
		msg = &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Caption:       proto.String(caption),
				Mimetype:      proto.String(mimetype),
				FileName:      proto.String(filename),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
			},
		}
	}

	resp, err := client.SendMessage(context.Background(), to, msg)
	if err != nil {
		bridgeLog.Warnf("send_media error: %v", err)
		return Response{Error: &ErrorInfo{Code: 1004, Message: "send failed"}}
	}

	return Response{Result: map[string]interface{}{
		"messageId": resp.ID,
	}}
}

func (s *Session) cmdSendReaction(cmd Command) Response {
	to, err := s.parseJID(cmd.Params, "to")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	messageId, _ := cmd.Params["messageId"].(string)
	emoji, _ := cmd.Params["emoji"].(string)

	if messageId == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'messageId' parameter"}}
	}

	resp, err := s.getClient().SendMessage(context.Background(), to, &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID: proto.String(to.String()),
				ID:        proto.String(messageId),
			},
			Text: proto.String(emoji),
		},
	})
	if err != nil {
		bridgeLog.Warnf("send_reaction error: %v", err)
		return Response{Error: &ErrorInfo{Code: 1004, Message: "reaction failed"}}
	}

	return Response{Result: map[string]interface{}{
		"messageId": resp.ID,
	}}
}

func (s *Session) cmdGetGroupInfo(cmd Command) Response {
	jid, err := s.parseJID(cmd.Params, "jid")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	info, err := s.getClient().GetGroupInfo(context.Background(), jid)
	if err != nil {
		bridgeLog.Warnf("get_group_info error: %v", err)
		return Response{Error: &ErrorInfo{Code: 1006, Message: "get group info failed"}}
	}

	participants := make([]map[string]interface{}, len(info.Participants))
	for i, p := range info.Participants {
		participants[i] = map[string]interface{}{
			"jid":          p.JID.String(),
			"isAdmin":      p.IsAdmin,
			"isSuperAdmin": p.IsSuperAdmin,
		}
	}

	return Response{Result: map[string]interface{}{
		"jid":          info.JID.String(),
		"name":         info.Name,
		"topic":        info.Topic,
		"participants": participants,
		"createdAt":    info.GroupCreated.Unix(),
	}}
}

func (s *Session) cmdSetPresence(cmd Command) Response {
	presenceType, _ := cmd.Params["type"].(string)

	client := s.getClient()
	switch presenceType {
	case "available":
		err := client.SendPresence(context.Background(), types.PresenceAvailable)
		if err != nil {
			bridgeLog.Warnf("set_presence error: %v", err)
			return Response{Error: &ErrorInfo{Code: 1007, Message: "failed to set presence"}}
		}
	case "unavailable":
		err := client.SendPresence(context.Background(), types.PresenceUnavailable)
		if err != nil {
			bridgeLog.Warnf("set_presence error: %v", err)
			return Response{Error: &ErrorInfo{Code: 1007, Message: "failed to set presence"}}
		}
	default:
		return Response{Error: &ErrorInfo{Code: 1003, Message: "type must be 'available' or 'unavailable'"}}
	}

	return Response{Result: map[string]interface{}{"ok": true}}
}