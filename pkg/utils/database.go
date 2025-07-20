package utils

import (
	"context"
	"crypto/rand"
	"database/sql"
	"firefly/pkg/protos"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oklog/ulid"
)

func OpenDatabase(connString string) (*pgxpool.Pool, error) {

	ctx := context.TODO()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	db, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Release()

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
CREATE OR REPLACE FUNCTION extract_ulid_timestamp(uuid_input UUID) RETURNS BIGINT AS $$
DECLARE
    uuid_hex TEXT;
    ulid_timestamp BIGINT;
BEGIN
    -- Convert UUID to hex text without dashes
    uuid_hex := replace(uuid_input::TEXT, '-', '');
    
    -- Extract the first 12 characters (6 bytes in hex) and convert to bigint
    ulid_timestamp := ('x' || substring(uuid_hex from 1 for 12))::BIT(48)::BIGINT;

    RETURN ulid_timestamp;
END;
$$ LANGUAGE plpgsql
			
		`)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
CREATE OR REPLACE FUNCTION generate_ulid() RETURNS uuid
    LANGUAGE plpgsql
    AS $$
DECLARE
  timestamp  BYTEA = E'\\000\\000\\000\\000\\000\\000';
  unix_time  BIGINT;
BEGIN
    unix_time = (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT;

    timestamp = SET_BYTE(timestamp, 0, (unix_time >> 40)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 1, (unix_time >> 32)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 2, (unix_time >> 24)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 3, (unix_time >> 16)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 4, (unix_time >> 8)::BIT(8)::INTEGER);
    timestamp = SET_BYTE(timestamp, 5, unix_time::BIT(8)::INTEGER);

    RETURN encode( timestamp || gen_random_bytes(10) ,'hex')::uuid;
END
$$
		`)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS groupchats (
    id SERIAL NOT NULL PRIMARY KEY,
    created_by VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
)
		`)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS groupchannels (
    chanId INTEGER NOT NULL,
    grpId INTEGER REFERENCES groupchats (id) NOT NULL,
    chanType SMALLINT NOT NULL,
)`)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS groupmembers (
    grpId INTEGER REFERENCES groupchats (id) NOT NULL,
    chanId INTEGER NOT NULL,
    uname VARCHAR NOT NULL,
    userrole INT NOT NULL DEFAULT 0,
    PRIMARY KEY (grpId, chanId, uname)
)`)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `CREATE INDEX IF NOT EXISTS uname_idx_on_groupmembers ON groupmembers (uname)`)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS groupchannels (
    chanId INTEGER NOT NULL,
    grpId INTEGER REFERENCES groupchats (id) NOT NULL,
    chanType SMALLINT NOT NULL,
)
`)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
CREATE INDEX IF NOT EXISTS grp_chan_index ON groupmessages (grpId, chanId);
`)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return pool, nil
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

func GetMessages(db *pgxpool.Pool, group_id int, channelId int, before uuid.UUID, count uint64) ([]*protos.GroupChannelMessage, error) {

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
