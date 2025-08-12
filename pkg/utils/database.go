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

	const STMT = "SELECT um.id, um.sender, um.msg FROM userchats uc WHERE uc.participant1 = $1 AND uc.participant2 = $2 JOIN usermessages um ON um.chatId = uc.id WHERE um.id < $3 LIMIT $4"

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
	rows, err := db.Query(context.TODO(), "SELECT id, participant1, participant2 FROM userchats WHERE participant1 = $1 OR participant2 = $1", username)
	var messages = &protos.UserMessages{
		Messages: make([]*protos.UserMessage, 0, 16),
	}
	if err != nil {
		return messages, err
	}

	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var other string
		var msg string

		err := rows.Scan(&id, &msg, &other)
		if err != nil {
			return messages, err
		}

		messages.Messages = append(messages.Messages, &protos.UserMessage{
			Id:   id[:],
			From: other,
			Text: msg,
			To:   username,
		})
	}
	return messages, nil
}
func InsertUserMessage(db *pgxpool.Pool, sender string, receiver string, msg string) (uuid.UUID, error) {

	senderFlag := sender > receiver

	row := db.QueryRow(context.TODO(), "INSERT INTO usermessages (sender, msg) VALUES ($1, $2) RETURNING id", senderFlag, msg)

	var id uuid.UUID
	err := row.Scan(&id)
	return id, err
}
