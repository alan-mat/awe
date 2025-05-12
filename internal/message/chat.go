package message

import (
	pb "github.com/alan-mat/awe/internal/proto"
)

type ChatRole int

const (
	RoleUser ChatRole = iota
	RoleAssistant
)

var roleName = map[ChatRole]string{
	RoleUser:      "user",
	RoleAssistant: "assistant",
}

func (r ChatRole) String() string {
	return roleName[r]
}

type Chat struct {
	Role    ChatRole
	Content string
}

func ParseChatHistory(h []*pb.ChatMessage) []*Chat {
	msgs := make([]*Chat, len(h))
	typesMap := map[pb.ChatRole]ChatRole{
		pb.ChatRole_UNSPECIFIED: RoleUser,
		pb.ChatRole_USER:        RoleUser,
		pb.ChatRole_ASSISTANT:   RoleAssistant,
	}
	for i, m := range h {
		chatmsg := &Chat{
			Role:    typesMap[m.GetRole()],
			Content: m.GetContent(),
		}
		msgs[i] = chatmsg
	}
	return msgs
}
