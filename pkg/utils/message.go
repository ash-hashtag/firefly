package utils

const (
	AuthorizationMsgType = iota
	ChatMessageMsgType
	AddMemberMsgType
	AddMemberToChannelMsgType
	KickMemberMsgType
	AddChannelMsgType
	DeleteChannelMsgType
	KickMemberOnlyFromChannelMsgType
	InfoMsgType
)

type Message struct {
	MsgType int    `json:"ty"`
	Payload string `json:"payload"`
	By      string `json:"by"`
}

type GroupChatMessage struct {
	ChannelName string `json:"channel"`
	Content     string `json:"payload"`
}
