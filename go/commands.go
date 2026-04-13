package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type commandHandler func(Command) Response

func (s *Session) buildCommandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"send_message":       s.cmdSendMessage,
		"send_image":         func(cmd Command) Response { return s.cmdSendMedia(cmd, "image") },
		"send_video":         func(cmd Command) Response { return s.cmdSendMedia(cmd, "video") },
		"send_audio":         func(cmd Command) Response { return s.cmdSendMedia(cmd, "audio") },
		"send_document":      func(cmd Command) Response { return s.cmdSendMedia(cmd, "document") },
		"send_reaction":      s.cmdSendReaction,
		"download_media":     s.cmdDownloadMedia,
		"get_group_info":     s.cmdGetGroupInfo,
		"send_chat_presence": s.cmdSendChatPresence,
		"mark_read":          s.cmdMarkRead,
		"set_presence":       s.cmdSetPresence,
	}
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

func (s *Session) buildContextInfo(params map[string]interface{}) *waE2E.ContextInfo {
	quotedId, _ := params["quotedId"].(string)
	quotedSender, _ := params["quotedSender"].(string)
	if quotedId == "" {
		return nil
	}
	return &waE2E.ContextInfo{
		StanzaID:    proto.String(quotedId),
		Participant: proto.String(quotedSender),
	}
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

	var msg *waE2E.Message
	if ci := s.buildContextInfo(cmd.Params); ci != nil {
		msg = &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(text),
				ContextInfo: ci,
			},
		}
	} else {
		msg = &waE2E.Message{
			Conversation: proto.String(text),
		}
	}

	bridgeLog.Debugf("[%s] send_message to=%s", s.name, to.String())
	resp, err := s.client.SendMessage(context.Background(), to, msg)
	if err != nil {
		bridgeLog.Errorf("[%s] send_message error: %v", s.name, err)
		return Response{Error: &ErrorInfo{Code: 1004, Message: "send failed"}}
	}

	return Response{Result: map[string]interface{}{
		"messageId": resp.ID,
	}}
}

var mediaUploadType = map[string]whatsmeow.MediaType{
	"image":    whatsmeow.MediaImage,
	"video":    whatsmeow.MediaVideo,
	"audio":    whatsmeow.MediaAudio,
	"document": whatsmeow.MediaDocument,
}

var mediaSizeLimit = map[string]int{
	"image":    16 * 1024 * 1024,
	"video":    16 * 1024 * 1024,
	"audio":    16 * 1024 * 1024,
	"document": 100 * 1024 * 1024,
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

	maxBytes := mediaSizeLimit[mediaType]

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
	contextInfo := s.buildContextInfo(cmd.Params)

	client := s.client
	uploaded, err := client.Upload(context.Background(), data, mediaUploadType[mediaType])
	if err != nil {
		bridgeLog.Warnf("[%s] upload error: %v", s.name, err)
		return Response{Error: &ErrorInfo{Code: 1005, Message: "media upload failed"}}
	}

	fileLen := proto.Uint64(uint64(len(data)))
	var msg *waE2E.Message

	switch mediaType {
	case "image":
		msg = &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:       proto.String(caption),
				Mimetype:      proto.String(mimetype),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    fileLen,
				ContextInfo:   contextInfo,
			},
		}
	case "video":
		msg = &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:       proto.String(caption),
				Mimetype:      proto.String(mimetype),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    fileLen,
				ContextInfo:   contextInfo,
			},
		}
	case "audio":
		msg = &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				Mimetype:      proto.String(mimetype),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    fileLen,
				ContextInfo:   contextInfo,
			},
		}
	case "document":
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
				FileLength:    fileLen,
				ContextInfo:   contextInfo,
			},
		}
	}

	resp, err := client.SendMessage(context.Background(), to, msg)
	if err != nil {
		bridgeLog.Warnf("[%s] send_media error: %v", s.name, err)
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

	resp, err := s.client.SendMessage(context.Background(), to, &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID: proto.String(to.String()),
				ID:        proto.String(messageId),
			},
			Text: proto.String(emoji),
		},
	})
	if err != nil {
		bridgeLog.Warnf("[%s] send_reaction error: %v", s.name, err)
		return Response{Error: &ErrorInfo{Code: 1004, Message: "reaction failed"}}
	}

	return Response{Result: map[string]interface{}{
		"messageId": resp.ID,
	}}
}

func (s *Session) cmdDownloadMedia(cmd Command) Response {
	ref, _ := cmd.Params["mediaRef"].(string)
	if ref == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'mediaRef' parameter"}}
	}

	destPath, _ := cmd.Params["path"].(string)
	if destPath == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'path' parameter"}}
	}

	// Prevent path traversal: destPath must resolve within the allowed download directory
	absBase, err := filepath.Abs(downloadDir)
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1009, Message: "server misconfiguration: invalid download dir"}}
	}
	absDest, err := filepath.Abs(destPath)
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "invalid path"}}
	}
	if !strings.HasPrefix(absDest+string(filepath.Separator), absBase+string(filepath.Separator)) {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "path outside allowed download directory"}}
	}

	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "invalid mediaRef format"}}
	}
	mediaType, encoded := parts[0], parts[1]

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "invalid mediaRef encoding"}}
	}

	var downloadable whatsmeow.DownloadableMessage
	switch mediaType {
	case "image":
		msg := &waE2E.ImageMessage{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return Response{Error: &ErrorInfo{Code: 1003, Message: "corrupt mediaRef"}}
		}
		downloadable = msg
	case "video":
		msg := &waE2E.VideoMessage{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return Response{Error: &ErrorInfo{Code: 1003, Message: "corrupt mediaRef"}}
		}
		downloadable = msg
	case "audio":
		msg := &waE2E.AudioMessage{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return Response{Error: &ErrorInfo{Code: 1003, Message: "corrupt mediaRef"}}
		}
		downloadable = msg
	case "document":
		msg := &waE2E.DocumentMessage{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return Response{Error: &ErrorInfo{Code: 1003, Message: "corrupt mediaRef"}}
		}
		downloadable = msg
	case "sticker":
		msg := &waE2E.StickerMessage{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return Response{Error: &ErrorInfo{Code: 1003, Message: "corrupt mediaRef"}}
		}
		downloadable = msg
	default:
		return Response{Error: &ErrorInfo{Code: 1003, Message: fmt.Sprintf("unsupported media type: %s", mediaType)}}
	}

	data, err := s.client.Download(context.Background(), downloadable)
	if err != nil {
		bridgeLog.Warnf("[%s] download_media error: %v", s.name, err)
		return Response{Error: &ErrorInfo{Code: 1009, Message: "media download failed"}}
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return Response{Error: &ErrorInfo{Code: 1009, Message: fmt.Sprintf("failed to create directory: %v", err)}}
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return Response{Error: &ErrorInfo{Code: 1009, Message: fmt.Sprintf("failed to write file: %v", err)}}
	}

	return Response{Result: map[string]interface{}{
		"path": destPath,
		"size": len(data),
	}}
}

func (s *Session) cmdGetGroupInfo(cmd Command) Response {
	jid, err := s.parseJID(cmd.Params, "jid")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	info, err := s.client.GetGroupInfo(context.Background(), jid)
	if err != nil {
		bridgeLog.Warnf("[%s] get_group_info error: %v", s.name, err)
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

func (s *Session) cmdSendChatPresence(cmd Command) Response {
	to, err := s.parseJID(cmd.Params, "to")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	state, _ := cmd.Params["state"].(string)
	media, _ := cmd.Params["media"].(string)

	var presence types.ChatPresence
	switch state {
	case "composing":
		presence = types.ChatPresenceComposing
	case "paused":
		presence = types.ChatPresencePaused
	default:
		return Response{Error: &ErrorInfo{Code: 1003, Message: "state must be 'composing' or 'paused'"}}
	}

	var mediaPresence types.ChatPresenceMedia
	switch media {
	case "audio":
		mediaPresence = types.ChatPresenceMediaAudio
	case "", "text":
		mediaPresence = types.ChatPresenceMediaText
	default:
		return Response{Error: &ErrorInfo{Code: 1003, Message: "media must be 'text' or 'audio'"}}
	}

	err = s.client.SendChatPresence(context.Background(), to, presence, mediaPresence)
	if err != nil {
		bridgeLog.Warnf("[%s] send_chat_presence error: %v", s.name, err)
		return Response{Error: &ErrorInfo{Code: 1007, Message: "failed to send chat presence"}}
	}

	return Response{Result: map[string]interface{}{"ok": true}}
}

func (s *Session) cmdMarkRead(cmd Command) Response {
	chat, err := s.parseJID(cmd.Params, "chat")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	sender, err := s.parseJID(cmd.Params, "sender")
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1003, Message: err.Error()}}
	}

	rawIds, ok := cmd.Params["messageIds"].([]interface{})
	if !ok || len(rawIds) == 0 {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing or empty 'messageIds' parameter"}}
	}

	ids := make([]types.MessageID, len(rawIds))
	for i, raw := range rawIds {
		id, ok := raw.(string)
		if !ok || id == "" {
			return Response{Error: &ErrorInfo{Code: 1003, Message: fmt.Sprintf("invalid message ID at index %d", i)}}
		}
		ids[i] = types.MessageID(id)
	}

	err = s.client.MarkRead(context.Background(), ids, time.Now(), chat, sender)
	if err != nil {
		bridgeLog.Warnf("[%s] mark_read error: %v", s.name, err)
		return Response{Error: &ErrorInfo{Code: 1008, Message: "failed to mark as read"}}
	}

	return Response{Result: map[string]interface{}{"ok": true}}
}

func (s *Session) cmdSetPresence(cmd Command) Response {
	presenceType, _ := cmd.Params["type"].(string)

	var presence types.Presence
	switch presenceType {
	case "available":
		presence = types.PresenceAvailable
	case "unavailable":
		presence = types.PresenceUnavailable
	default:
		return Response{Error: &ErrorInfo{Code: 1003, Message: "type must be 'available' or 'unavailable'"}}
	}

	if err := s.client.SendPresence(context.Background(), presence); err != nil {
		bridgeLog.Warnf("[%s] set_presence error: %v", s.name, err)
		return Response{Error: &ErrorInfo{Code: 1007, Message: "failed to set presence"}}
	}

	return Response{Result: map[string]interface{}{"ok": true}}
}
