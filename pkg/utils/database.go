package utils

import (
	"context"
	"crypto/rand"
	"database/sql"
	"firefly/pkg/protos"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oklog/ulid"
)

func OpenDatabase(connString string) (*pgxpool.Pool, error) {
	ctx := context.TODO()

	poolSizeStr := os.Getenv("DB_POOL_SIZE")

	var maxConns = int32(100)
	if len(poolSizeStr) > 0 {
		poolSize, err := strconv.ParseInt(poolSizeStr, 10, 32)
		if err != nil {
			return nil, err
		}
		if poolSize > 0 && poolSize < 1000 {
			maxConns = int32(poolSize)
		}
	}

	connConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	connConfig.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(ctx, connConfig)
	return pool, err
}

func NewUlid(timestamp uint64) ulid.ULID {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(timestamp, entropy)
}

func MinULIDAt(timestamp uint64) ulid.ULID {
	id := NewUlid(timestamp)
	copy(id[6:], make([]byte, 10))
	return id
}

func MaxULIDAt(timestamp uint64) ulid.ULID {
	id := NewUlid(timestamp)
	for i := 6; i < 16; i++ {
		id[i] = 255
	}
	return id

}

func GetGroupMessages(db *pgxpool.Pool, group_id int, channelId int, before uuid.UUID, count uint64) ([]*protos.GroupChannelMessage, error) {

	const stmt = "SELECT id, by, msg FROM groupmessages WHERE grpId = $1 AND chanId = $2 AND id < $3 LIMIT $4"
	rows, err := db.Query(context.TODO(), stmt, before, count)
	if err != nil {
		return nil, err
	}

	messages := make([]*protos.GroupChannelMessage, 0, 512)

	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var by, msg string
		err := rows.Scan(&id, &by, &msg)
		if err != nil {
			return nil, err
		}
		message := protos.GroupChannelMessage{}

		message.Id = id[:]
		message.ChannelId = int32(channelId)
		message.Content = msg
		message.By = by
		messages = append(messages, &message)
	}

	return messages, nil
}

func GetUserMessages(db *pgxpool.Pool, participant1 string, participant2 string, before uuid.UUID, count uint64) (*protos.UserMessages, error) {
	var higher string
	var lower string
	if participant1 > participant2 {
		higher = participant1
		lower = participant2
	} else {
		higher = participant2
		lower = participant1
	}

	const STMT = "SELECT um.id, um.sender, um.msg FROM userchats uc JOIN usermessages um ON um.chatId = uc.id WHERE uc.participant1 = $1 AND uc.participant2 = $2 AND um.id < $3 LIMIT $4"

	rows, err := db.Query(context.TODO(), STMT, higher, lower, before, count)
	if err != nil {
		return nil, err
	}

	messages := make([]*protos.UserMessage, 0, 512)

	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var msg string
		var sender bool
		err := rows.Scan(&id, &sender, &msg)
		if err != nil {
			return nil, err
		}
		message := protos.UserMessage{}

		message.Id = id[:]
		if sender {
			message.From = higher
			message.To = lower
		} else {
			message.From = lower
			message.To = higher
		}

		message.Text = msg
		messages = append(messages, &message)
	}

	return &protos.UserMessages{Messages: messages}, nil
}

func AddChannelMessage(db *pgxpool.Pool, msg *protos.GroupChannelMessage) (*protos.GroupChannelMessage, error) {
	stmt := "INSERT INTO groupmessages (id, grpId, chanId, msg, by) VALUES ($1, $2, $3, $4, $5)"

	id := uuid.UUID([16]byte(NewUlid(ulid.Now())))

	_, err := db.Exec(context.TODO(), stmt, id, msg.GetGroupId(), msg.GetChannelId(), msg.GetContent(), msg.GetBy())

	msg.Id = id[:]

	return msg, err
}

func DeleteChannel(db *pgxpool.Pool, groupId int, channelId int) error {
	_, err := db.Exec(context.TODO(), "DELETE FROM groupchannels WHERE grpId = $1 AND chanId = $2", groupId, channelId)

	return err

}

func CreateChannel(db *pgxpool.Pool, groupId int, channelName string, channelType int, createdBy string) (int, error) {
	row := db.QueryRow(context.TODO(), "SELECT create_group_channel($1, $2, $3, $4)", groupId, channelType, channelName, createdBy)

	var channelId int
	err := row.Scan(&channelId)
	return channelId, err

}

func GetOwner(db *pgxpool.Pool, groupId int) (string, error) {
	row := db.QueryRow(context.TODO(), "SELECT created_by FROM groupchats WHERE id = $1", groupId)

	var creator string
	err := row.Scan(&creator)

	return creator, err

}

const (
	ChannelText = iota
	ChannelVoice
	ChannelVideo
)

func RemoveUserFromEverything(db *sql.DB, username string, groupId int) error {
	stmt := "DELETE FROM groupmembers WHERE grpId = $1 AND uname = $2"
	_, err := db.Exec(stmt, groupId, username)
	return err
}

func RemoveUserFromChannel(db *pgxpool.Pool, username string, groupId, channelId int, beingRemovedBy string) error {
	const STMT = "SELECT kick_group_member($1, $2, $3, $4)"

	_, err := db.Exec(context.TODO(), STMT, username, groupId, channelId, beingRemovedBy)

	return err
}

// / if channel is -1, user will be added to all the channels `being_added_by` can add
func AddUserToChannel(db *pgxpool.Pool, username string, groupId int, channelId int, role int, beingAddedBy string) error {

	_, err := db.Exec(context.TODO(), "SELECT add_group_member ($1, $2, $3, $4, $5)", username, groupId, channelId, role, beingAddedBy)

	return err

}

func GetUsersInChannel(db *pgxpool.Pool, groupId int, channelId int) ([]*protos.GroupMember, error) {

	rows, err := db.Query(context.TODO(), "SELECT uname, role FROM groupmembers WHERE grpId = $1 AND chanId = $2", groupId, channelId)
	users := make([]*protos.GroupMember, 0, 100)
	if err != nil {
		return users, err
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var role int
		rows.Scan(&username, &role)
		user := protos.GroupMember{}
		user.Username = username
		user.Role = int32(role)
		user.ChanId = int32(channelId)
		users = append(users, &user)
	}
	return users, nil
}

func GetAllUsers(db *pgxpool.Pool, groupId int) ([]*protos.GroupMember, error) {
	rows, err := db.Query(context.TODO(), "SELECT uname, userrole, chanId FROM groupmembers WHERE grpId = $1", groupId)

	users := make([]*protos.GroupMember, 0, 512)
	if err != nil {
		return users, err
	}

	defer rows.Close()

	for rows.Next() {
		var username string
		var role int
		var chanId int
		err := rows.Scan(&username, &role, &chanId)
		if err != nil {
			return users, err
		}

		user := protos.GroupMember{}
		user.Username = username
		user.Role = int32(role)
		user.ChanId = int32(chanId)

		users = append(users, &user)
	}

	return users, nil
}

func GetChannelsUserHasAccessTo(db *pgxpool.Pool, grpId int, username string) ([]*protos.GroupChannel, error) {
	channels := make([]*protos.GroupChannel, 0, 20)
	rows, err := db.Query(context.TODO(), "SELECT gc.chanId, gc.name, gc.chanType FROM groupmembers gm JOIN groupchannels gc ON gm.grpId = gc.grpId WHERE gm.grpId = $1 AND gm.uname = $2", grpId, username)

	if err != nil {
		return channels, err
	}

	defer rows.Close()

	for rows.Next() {
		var chanId int
		var name string
		var chanType int

		err := rows.Scan(&chanId, &name, &chanType)
		if err != nil {
			return channels, err
		}
		channel := protos.GroupChannel{
			ChannelType: int32(chanType),
			Name:        name,
			ChannelId:   int32(chanId),
		}

		channels = append(channels, &channel)
	}

	return channels, nil

}

func GetAllChannels(db *pgxpool.Pool, grpId int) ([]*protos.GroupChannel, error) {
	channels := make([]*protos.GroupChannel, 0, 20)
	rows, err := db.Query(context.TODO(), "SELECT chanId, name, chanType FROM groupchannels WHERE grpId = $1", grpId)

	if err != nil {
		return channels, err
	}

	defer rows.Close()

	for rows.Next() {
		var chanId int
		var name string
		var chanType int

		err := rows.Scan(&chanId, &name, &chanType)
		if err != nil {
			return channels, err
		}
		channel := protos.GroupChannel{
			ChannelType: int32(chanType),
			Name:        name,
			ChannelId:   int32(chanId),
		}

		channels = append(channels, &channel)
	}

	return channels, nil
}

func GetAllGroupChatsUserIn(db *pgxpool.Pool, username string) (*protos.GroupChats, error) {
	rows, err := db.Query(context.TODO(), "SELECT UNIQUE(gm.grpId), gc.name, FROM groupmembers gm JOIN groupchats gc ON gc.id = gm.grpId WHERE gm.uname = $1", username)
	var groups = &protos.GroupChats{
		Chats: make([]*protos.GroupChat, 0),
	}
	if err != nil {
		return groups, err
	}
	defer rows.Close()

	for rows.Next() {
		var groupId int
		var groupName string
		err := rows.Scan(&groupId, &groupName)
		if err != nil {
			return groups, err
		}

		chat := protos.GroupChat{
			Name:    groupName,
			GroupId: int32(groupId),
		}

		groups.Chats = append(groups.Chats, &chat)

	}

	return groups, nil
}

func GetAllUserChats(db *pgxpool.Pool, username string) (*protos.UserMessages, error) {

	// 	const STMT = `
	// SELECT  uc.id,
	// 		uc.participant1,
	// 		uc.participant2,
	// 		lm.id,
	//         COALESCE(lm.sender, false),
	//         COALESCE(lm.msg, '')
	// FROM userchats uc
	// LEFT JOIN (
	//     SELECT DISTINCT ON (um.chatId) um.*
	//     FROM usermessages um
	//     ORDER BY um.chatId, um.id DESC
	// ) lm
	// ON lm.chatId = uc.id
	// WHERE uc.participant1 = $1 OR uc.participant2 = $1
	// 		`

	const STMT = "SELECT * FROM get_userchats($1)"

	rows, err := db.Query(context.TODO(), STMT, username)
	var messages = &protos.UserMessages{
		Messages: make([]*protos.UserMessage, 0, 16),
	}
	if err != nil {
		return messages, err
	}

	defer rows.Close()

	for rows.Next() {
		var chatId, msgId uuid.UUID
		var senderFlag bool
		var participant1, participant2, msg string

		err := rows.Scan(&chatId, &participant1, &participant2, &msgId, &senderFlag, &msg)
		if err != nil {
			return messages, err
		}

		var from string
		var to string

		if senderFlag {
			from = participant1
			to = participant2
		} else {
			from = participant2
			to = participant1
		}

		messages.Messages = append(messages.Messages, &protos.UserMessage{
			Id:   msgId[:],
			From: from,
			Text: msg,
			To:   to,
		})
	}
	return messages, nil
}
func InsertUserMessage(db *pgxpool.Pool, sender string, receiver string, msg string) (uuid.UUID, error) {

	var higher, lower string
	var senderFlag bool
	if sender > receiver {
		higher = sender
		lower = receiver
		senderFlag = true
	} else {
		lower = sender
		higher = receiver
		senderFlag = false
	}

	const STMT = "INSERT INTO usermessages (sender, msg, chatId) SELECT $1, $2, uc.id FROM userchats uc WHERE uc.participant1 = $3 AND uc.participant2 = $4 RETURNING id"

	row := db.QueryRow(context.TODO(), STMT, senderFlag, msg, higher, lower)

	var id uuid.UUID
	err := row.Scan(&id)
	return id, err
}

func CreateGroupChat(db *pgxpool.Pool, groupName string, owner string) (int, error) {

	row := db.QueryRow(context.TODO(), "INSERT INTO groupchats (name, created_by) VALUES ($1, $2) RETURNING id", groupName, owner)
	var groupId int

	err := row.Scan(&groupId)

	return groupId, err
}

func CreateUserChat(db *pgxpool.Pool, participant1, participant2 string) (uuid.UUID, error) {

	var higher, lower string

	if participant1 > participant2 {
		higher = participant1
		lower = participant2
	} else {
		lower = participant1
		higher = participant2
	}

	const STMT = "INSERT INTO userchats (participant1, participant2) VALUES ($1, $2) ON CONFLICT (participant1, participant2) DO UPDATE SET participant1=EXCLUDED.participant1 RETURNING id"

	row := db.QueryRow(context.TODO(), STMT, higher, lower)
	var chatId uuid.UUID
	err := row.Scan(&chatId)

	return chatId, err

}
