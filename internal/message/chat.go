package message

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
