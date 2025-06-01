// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

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
