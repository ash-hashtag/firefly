// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.29.2
// source: message.proto

package utils

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type FDBTextMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Msg           string                 `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FDBTextMessage) Reset() {
	*x = FDBTextMessage{}
	mi := &file_message_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FDBTextMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FDBTextMessage) ProtoMessage() {}

func (x *FDBTextMessage) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FDBTextMessage.ProtoReflect.Descriptor instead.
func (*FDBTextMessage) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{0}
}

func (x *FDBTextMessage) GetMsg() string {
	if x != nil {
		return x.Msg
	}
	return ""
}

type MarkdownMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Msg           []byte                 `protobuf:"bytes,1,opt,name=msg,proto3" json:"msg,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MarkdownMessage) Reset() {
	*x = MarkdownMessage{}
	mi := &file_message_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MarkdownMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MarkdownMessage) ProtoMessage() {}

func (x *MarkdownMessage) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MarkdownMessage.ProtoReflect.Descriptor instead.
func (*MarkdownMessage) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{1}
}

func (x *MarkdownMessage) GetMsg() []byte {
	if x != nil {
		return x.Msg
	}
	return nil
}

type VersionedMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Version       uint32                 `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Ts            uint64                 `protobuf:"varint,2,opt,name=ts,proto3" json:"ts,omitempty"`
	Data          []byte                 `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *VersionedMessage) Reset() {
	*x = VersionedMessage{}
	mi := &file_message_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *VersionedMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VersionedMessage) ProtoMessage() {}

func (x *VersionedMessage) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VersionedMessage.ProtoReflect.Descriptor instead.
func (*VersionedMessage) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{2}
}

func (x *VersionedMessage) GetVersion() uint32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *VersionedMessage) GetTs() uint64 {
	if x != nil {
		return x.Ts
	}
	return 0
}

func (x *VersionedMessage) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type GroupMessage struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Message:
	//
	//	*GroupMessage_AddUser
	//	*GroupMessage_RemoveUser
	//	*GroupMessage_GroupMessage
	//	*GroupMessage_AuthToken
	Message       isGroupMessage_Message `protobuf_oneof:"message"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GroupMessage) Reset() {
	*x = GroupMessage{}
	mi := &file_message_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GroupMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupMessage) ProtoMessage() {}

func (x *GroupMessage) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupMessage.ProtoReflect.Descriptor instead.
func (*GroupMessage) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{3}
}

func (x *GroupMessage) GetMessage() isGroupMessage_Message {
	if x != nil {
		return x.Message
	}
	return nil
}

func (x *GroupMessage) GetAddUser() *AddUser {
	if x != nil {
		if x, ok := x.Message.(*GroupMessage_AddUser); ok {
			return x.AddUser
		}
	}
	return nil
}

func (x *GroupMessage) GetRemoveUser() *RemoveUser {
	if x != nil {
		if x, ok := x.Message.(*GroupMessage_RemoveUser); ok {
			return x.RemoveUser
		}
	}
	return nil
}

func (x *GroupMessage) GetGroupMessage() *GroupChannelMessage {
	if x != nil {
		if x, ok := x.Message.(*GroupMessage_GroupMessage); ok {
			return x.GroupMessage
		}
	}
	return nil
}

func (x *GroupMessage) GetAuthToken() *AuthenticationToken {
	if x != nil {
		if x, ok := x.Message.(*GroupMessage_AuthToken); ok {
			return x.AuthToken
		}
	}
	return nil
}

type isGroupMessage_Message interface {
	isGroupMessage_Message()
}

type GroupMessage_AddUser struct {
	AddUser *AddUser `protobuf:"bytes,1,opt,name=addUser,proto3,oneof"`
}

type GroupMessage_RemoveUser struct {
	RemoveUser *RemoveUser `protobuf:"bytes,2,opt,name=removeUser,proto3,oneof"`
}

type GroupMessage_GroupMessage struct {
	GroupMessage *GroupChannelMessage `protobuf:"bytes,3,opt,name=groupMessage,proto3,oneof"`
}

type GroupMessage_AuthToken struct {
	AuthToken *AuthenticationToken `protobuf:"bytes,4,opt,name=authToken,proto3,oneof"`
}

func (*GroupMessage_AddUser) isGroupMessage_Message() {}

func (*GroupMessage_RemoveUser) isGroupMessage_Message() {}

func (*GroupMessage_GroupMessage) isGroupMessage_Message() {}

func (*GroupMessage_AuthToken) isGroupMessage_Message() {}

type AuthenticationToken struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Token         string                 `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AuthenticationToken) Reset() {
	*x = AuthenticationToken{}
	mi := &file_message_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AuthenticationToken) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthenticationToken) ProtoMessage() {}

func (x *AuthenticationToken) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthenticationToken.ProtoReflect.Descriptor instead.
func (*AuthenticationToken) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{4}
}

func (x *AuthenticationToken) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

type AddUser struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Username      string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	ToChannel     string                 `protobuf:"bytes,2,opt,name=toChannel,proto3" json:"toChannel,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddUser) Reset() {
	*x = AddUser{}
	mi := &file_message_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddUser) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddUser) ProtoMessage() {}

func (x *AddUser) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddUser.ProtoReflect.Descriptor instead.
func (*AddUser) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{5}
}

func (x *AddUser) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *AddUser) GetToChannel() string {
	if x != nil {
		return x.ToChannel
	}
	return ""
}

type RemoveUser struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Username      string                 `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	FromChannel   string                 `protobuf:"bytes,2,opt,name=fromChannel,proto3" json:"fromChannel,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RemoveUser) Reset() {
	*x = RemoveUser{}
	mi := &file_message_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RemoveUser) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoveUser) ProtoMessage() {}

func (x *RemoveUser) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoveUser.ProtoReflect.Descriptor instead.
func (*RemoveUser) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{6}
}

func (x *RemoveUser) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *RemoveUser) GetFromChannel() string {
	if x != nil {
		return x.FromChannel
	}
	return ""
}

type GroupChannelMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            []byte                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	InChannel     int32                  `protobuf:"varint,2,opt,name=inChannel,proto3" json:"inChannel,omitempty"`
	Content       string                 `protobuf:"bytes,3,opt,name=content,proto3" json:"content,omitempty"`
	By            string                 `protobuf:"bytes,4,opt,name=by,proto3" json:"by,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GroupChannelMessage) Reset() {
	*x = GroupChannelMessage{}
	mi := &file_message_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GroupChannelMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupChannelMessage) ProtoMessage() {}

func (x *GroupChannelMessage) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupChannelMessage.ProtoReflect.Descriptor instead.
func (*GroupChannelMessage) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{7}
}

func (x *GroupChannelMessage) GetId() []byte {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *GroupChannelMessage) GetInChannel() int32 {
	if x != nil {
		return x.InChannel
	}
	return 0
}

func (x *GroupChannelMessage) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *GroupChannelMessage) GetBy() string {
	if x != nil {
		return x.By
	}
	return ""
}

type GroupChannelMessages struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Messages      []*GroupChannelMessage `protobuf:"bytes,1,rep,name=messages,proto3" json:"messages,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GroupChannelMessages) Reset() {
	*x = GroupChannelMessages{}
	mi := &file_message_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GroupChannelMessages) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupChannelMessages) ProtoMessage() {}

func (x *GroupChannelMessages) ProtoReflect() protoreflect.Message {
	mi := &file_message_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupChannelMessages.ProtoReflect.Descriptor instead.
func (*GroupChannelMessages) Descriptor() ([]byte, []int) {
	return file_message_proto_rawDescGZIP(), []int{8}
}

func (x *GroupChannelMessages) GetMessages() []*GroupChannelMessage {
	if x != nil {
		return x.Messages
	}
	return nil
}

var File_message_proto protoreflect.FileDescriptor

var file_message_proto_rawDesc = string([]byte{
	0x0a, 0x0d, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0a, 0x6c, 0x75, 0x70, 0x79, 0x64, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x22, 0x22, 0x0a, 0x0e, 0x46,
	0x44, 0x42, 0x54, 0x65, 0x78, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x10, 0x0a,
	0x03, 0x6d, 0x73, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6d, 0x73, 0x67, 0x22,
	0x23, 0x0a, 0x0f, 0x4d, 0x61, 0x72, 0x6b, 0x64, 0x6f, 0x77, 0x6e, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x03, 0x6d, 0x73, 0x67, 0x22, 0x50, 0x0a, 0x10, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x65,
	0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02,
	0x74, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x8c, 0x02, 0x0a, 0x0c, 0x47, 0x72, 0x6f, 0x75, 0x70,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x2f, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x55, 0x73,
	0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x6c, 0x75, 0x70, 0x79, 0x64,
	0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x41, 0x64, 0x64, 0x55, 0x73, 0x65, 0x72, 0x48, 0x00, 0x52,
	0x07, 0x61, 0x64, 0x64, 0x55, 0x73, 0x65, 0x72, 0x12, 0x38, 0x0a, 0x0a, 0x72, 0x65, 0x6d, 0x6f,
	0x76, 0x65, 0x55, 0x73, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x6c,
	0x75, 0x70, 0x79, 0x64, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65,
	0x55, 0x73, 0x65, 0x72, 0x48, 0x00, 0x52, 0x0a, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x55, 0x73,
	0x65, 0x72, 0x12, 0x45, 0x0a, 0x0c, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6c, 0x75, 0x70, 0x79, 0x64,
	0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x48, 0x00, 0x52, 0x0c, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x3f, 0x0a, 0x09, 0x61, 0x75, 0x74,
	0x68, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6c,
	0x75, 0x70, 0x79, 0x64, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e,
	0x74, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x48, 0x00, 0x52,
	0x09, 0x61, 0x75, 0x74, 0x68, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x09, 0x0a, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x2b, 0x0a, 0x13, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74,
	0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x14, 0x0a, 0x05,
	0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b,
	0x65, 0x6e, 0x22, 0x43, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x55, 0x73, 0x65, 0x72, 0x12, 0x1a, 0x0a,
	0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x6f, 0x43,
	0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x6f,
	0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x22, 0x4a, 0x0a, 0x0a, 0x52, 0x65, 0x6d, 0x6f, 0x76,
	0x65, 0x55, 0x73, 0x65, 0x72, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x20, 0x0a, 0x0b, 0x66, 0x72, 0x6f, 0x6d, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x66, 0x72, 0x6f, 0x6d, 0x43, 0x68, 0x61, 0x6e,
	0x6e, 0x65, 0x6c, 0x22, 0x6d, 0x0a, 0x13, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x6e,
	0x6e, 0x65, 0x6c, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x69, 0x6e,
	0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x69,
	0x6e, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74,
	0x65, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65,
	0x6e, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x62, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02,
	0x62, 0x79, 0x22, 0x53, 0x0a, 0x14, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x12, 0x3b, 0x0a, 0x08, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6c,
	0x75, 0x70, 0x79, 0x64, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x43,
	0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x08, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x42, 0x09, 0x5a, 0x07, 0x2e, 0x2f, 0x75, 0x74, 0x69,
	0x6c, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_message_proto_rawDescOnce sync.Once
	file_message_proto_rawDescData []byte
)

func file_message_proto_rawDescGZIP() []byte {
	file_message_proto_rawDescOnce.Do(func() {
		file_message_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_message_proto_rawDesc), len(file_message_proto_rawDesc)))
	})
	return file_message_proto_rawDescData
}

var file_message_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_message_proto_goTypes = []any{
	(*FDBTextMessage)(nil),       // 0: lupyd.chat.FDBTextMessage
	(*MarkdownMessage)(nil),      // 1: lupyd.chat.MarkdownMessage
	(*VersionedMessage)(nil),     // 2: lupyd.chat.VersionedMessage
	(*GroupMessage)(nil),         // 3: lupyd.chat.GroupMessage
	(*AuthenticationToken)(nil),  // 4: lupyd.chat.AuthenticationToken
	(*AddUser)(nil),              // 5: lupyd.chat.AddUser
	(*RemoveUser)(nil),           // 6: lupyd.chat.RemoveUser
	(*GroupChannelMessage)(nil),  // 7: lupyd.chat.GroupChannelMessage
	(*GroupChannelMessages)(nil), // 8: lupyd.chat.GroupChannelMessages
}
var file_message_proto_depIdxs = []int32{
	5, // 0: lupyd.chat.GroupMessage.addUser:type_name -> lupyd.chat.AddUser
	6, // 1: lupyd.chat.GroupMessage.removeUser:type_name -> lupyd.chat.RemoveUser
	7, // 2: lupyd.chat.GroupMessage.groupMessage:type_name -> lupyd.chat.GroupChannelMessage
	4, // 3: lupyd.chat.GroupMessage.authToken:type_name -> lupyd.chat.AuthenticationToken
	7, // 4: lupyd.chat.GroupChannelMessages.messages:type_name -> lupyd.chat.GroupChannelMessage
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_message_proto_init() }
func file_message_proto_init() {
	if File_message_proto != nil {
		return
	}
	file_message_proto_msgTypes[3].OneofWrappers = []any{
		(*GroupMessage_AddUser)(nil),
		(*GroupMessage_RemoveUser)(nil),
		(*GroupMessage_GroupMessage)(nil),
		(*GroupMessage_AuthToken)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_message_proto_rawDesc), len(file_message_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_message_proto_goTypes,
		DependencyIndexes: file_message_proto_depIdxs,
		MessageInfos:      file_message_proto_msgTypes,
	}.Build()
	File_message_proto = out.File
	file_message_proto_goTypes = nil
	file_message_proto_depIdxs = nil
}
