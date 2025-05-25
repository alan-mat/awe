package api

import (
	pb "github.com/alan-mat/awe/internal/proto"
)

type ChatMessageRole int

const (
	RoleUser ChatMessageRole = iota
	RoleAssistant
)

var roleName = map[ChatMessageRole]string{
	RoleUser:      "user",
	RoleAssistant: "assistant",
}

func (r ChatMessageRole) String() string {
	return roleName[r]
}

type ChatMessage struct {
	Role    ChatMessageRole
	Content string
}

func ParseChatHistory(h []*pb.ChatMessage) []*ChatMessage {
	msgs := make([]*ChatMessage, len(h))
	typesMap := map[pb.ChatRole]ChatMessageRole{
		pb.ChatRole_ROLE_UNSPECIFIED: RoleUser,
		pb.ChatRole_USER:             RoleUser,
		pb.ChatRole_ASSISTANT:        RoleAssistant,
	}
	for i, m := range h {
		chatmsg := &ChatMessage{
			Role:    typesMap[m.GetRole()],
			Content: m.GetContent(),
		}
		msgs[i] = chatmsg
	}
	return msgs
}
