package utils

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/oklog/ulid"
)

func Check(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func CreateChannelMessagesTable(db *sql.DB, channel_id int) {

	table_name := ChannelMessagesTableName(channel_id)

	stmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id UHUGEINT PRIMARY KEY, by VARCHAR NOT NULL, msg VARCHAR NOT NULL)", table_name)
	_, err := db.Exec(stmt)
	Check(err)

	log.Printf("Created Channel Messages Table %s", table_name)
}

func CreateChannelsTable(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS channels (id INT NOT NULL, name VARCHAR NOT NULL PRIMARY KEY, ty TINYINT NOT NULL)")
	Check(err)
}

func OpenDatabase(filepath string, creator string) *sql.DB {
	db, err := sql.Open("duckdb", filepath)
	Check(err)

	CreateUsersTable(db)
	CreateHigherPriviligedUsersTable(db)
	CreateChannelsTable(db)

	AddHigherPriviledUser(db, "server", RoleServer)
	AddHigherPriviledUser(db, creator, RoleOwner)

	rows, err := db.Query("SELECT id, name, ty FROM channels")
	Check(err)
	for rows.Next() {
		var channel_id int
		var channel_name string
		var channel_ty int
		Check(rows.Scan(&channel_id, &channel_name, &channel_ty))
		CreateChannel(db, channel_name, channel_ty)
	}

	return db
}

func CreateHigherPriviligedUsersTable(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS priviliged_users (username VARCHAR PRIMARY KEY, role TINYINT NOT NULL)")
	Check(err)

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS priviliged_username_idx ON priviliged_users ( username )")
	Check(err)

}

func AddHigherPriviledUser(db *sql.DB, username string, role int) {
	_, err := db.Exec("INSERT OR REPLACE INTO priviliged_users VALUES ($1, $2)", username, role)
	Check(err)
}

func RemoveHigherPriviligedUser(db *sql.DB, username string) {
	_, err := db.Exec("DELETE FROM priviliged_users WHERE username = $1", username)
	Check(err)
}

func AddMessage(db *sql.DB, by, msg string, channel_id int) (ulid.ULID, error) {

	if GetUserRole(db, by, channel_id) == -1 {
		return ulid.ULID{}, errors.New("User can't access this channel")
	}

	msg_id := NewUlid(ulid.Now())
	msg_id_as_big_int := new(big.Int).SetBytes(msg_id[:])
	stmt := fmt.Sprintf("INSERT INTO channel_messages_%d VALUES ($1, $2, $3)", channel_id)
	_, err := db.Exec(stmt, msg_id_as_big_int, by, msg)
	return msg_id, err
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

func GetMessages(db *sql.DB, channel_id int, start_at uint64, end_at uint64) ([]*GroupChannelMessage, error) {
	stmt := fmt.Sprintf("SELECT id, by, msg FROM %s WHERE id >= $1 AND id <= $2", ChannelMessagesTableName(channel_id))
	rows, err := db.Query(stmt, MinULIDAt(start_at), MaxULIDAt(end_at))
	if err != nil {
		return nil, err
	}

	messages := make([]*GroupChannelMessage, 0, 512)

	for rows.Next() {
		var id *big.Int
		var by, msg string
		Check(rows.Scan(id, &by, &msg))
		message := GroupChannelMessage{}
		message.Id = id.Bytes()
		message.InChannel = int32(channel_id)
		message.Content = msg
		message.By = by
		messages = append(messages, &message)
	}

	return messages, nil
}

func ChannelMessagesTableName(channel_id int) string {
	return fmt.Sprintf("channel_messages_%d", channel_id)
}

func DeleteChannel(db *sql.DB, channel_id int) error {
	result, err := db.Exec("DELETE FROM channels WHERE id = $1", channel_id)

	if err != nil {
		return err
	}

	rows_affected, err := result.RowsAffected()

	Check(err)

	if rows_affected == 0 {
		return fmt.Errorf("No channel with id %d", channel_id)
	}

	stmt := fmt.Sprintf("DROP TABLE %s", ChannelMessagesTableName(channel_id))
	_, err = db.Exec(stmt)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM users WHERE channel_id = $1", channel_id)

	return err
}

func CreateChannel(db *sql.DB, channel_name string, channel_type int) (int, error) {

	rand_id, err := rand.Int(rand.Reader, big.NewInt(int64(^uint16(0))))
	Check(err)

	var channel_id int

	row := db.QueryRow("SELECT id FROM channels WHERE name = $1", channel_name)

	if row.Scan(&channel_id) == nil {
		return channel_id, nil
	}

	channel_id = int(rand_id.Int64())
	_, err = db.Exec("INSERT INTO channels (id, name, ty) VALUES ( $1, $2, $3 )", channel_id, channel_name, channel_type)
	Check(err)
	log.Printf("Created Channel '%s' with id: %d", channel_name, channel_id)
	CreateChannelMessagesTable(db, channel_id)
	return channel_id, nil
}

func GetOwner(db *sql.DB) string {
	row := db.QueryRow("SELECT username FROM priviliged_users WHERE role = $1", RoleOwner)

	var owner_username string
	Check(row.Scan(&owner_username))

	return owner_username

}

type UserWithRole = struct {
	username string
	role     int
}

type UserWithRoleInChannel = struct {
	username   string
	role       int
	channel_id int
}

func GetPriviligedUsers(db *sql.DB) []UserWithRole {
	users := make([]UserWithRole, 0, 32)
	rows, err := db.Query("SELECT username, role FROM priviliged_users")
	Check(err)
	for rows.Next() {
		var role int
		var username string
		rows.Scan(&username, &role)
		users = append(users, UserWithRole{username: username, role: role})
	}

	return users
}

func GetAdminAllUsers(db *sql.DB) []string {
	rows, err := db.Query("SELECT username FROM users WHERE role = $1", RoleAdminAll)
	Check(err)

	users := make([]string, 0, 64)
	for rows.Next() {
		var username string
		rows.Scan(&username)
		users = append(users, username)
	}
	return users
}

const (
	RoleUser = iota
	RoleAdmin
	RoleAdminAll
	RoleOwner
	RoleServer
)

const (
	ChannelText = iota
	ChannelVoice
	ChannelVideo
)

func CreateUsersTable(db *sql.DB) {
	stmt := "CREATE TABLE IF NOT EXISTS users ( username VARCHAR NOT NULL, role TINYINT NOT NULL, channel_id INT NOT NULL, UNIQUE(username, channel_id) )"
	_, err := db.Exec(stmt)
	Check(err)
	log.Printf("Created users Table")
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS username_idx ON users ( username )")
	Check(err)
}

func RemoveUserFromEverything(db *sql.DB, username string) {
	stmt := "DELETE FROM users WHERE username = $1"
	_, err := db.Exec(stmt, username)
	Check(err)
	RemoveHigherPriviligedUser(db, username)
	log.Printf("Deleted User %s from every channel and group", username)
}

func RemoveUserFromChannel(db *sql.DB, username string, channel_id int) {
	stmt := "DELETE FROM users WHERE username = $1 AND channel_id = $2"
	_, err := db.Exec(stmt, username, channel_id)
	Check(err)
	log.Printf("Deleted User %s from channel %d", username, channel_id)
}

func GetUserRole(db *sql.DB, username string, channel_id int) int {

	row := db.QueryRow("SELECT role FROM priviliged_users WHERE username = $1 ", username)
	var role int8
	if row.Scan(&role) == nil {
		return int(role)
	}

	row = db.QueryRow("SELECT role FROM users WHERE username = $1 AND channel_id = $2", username, channel_id)

	if row.Scan(&role) == nil {
		return int(role)
	}

	return -1
}

func AddUserToChannel(db *sql.DB, username string, user_role int, channel_id int, being_added_by string) error {
	var being_added_by_role = GetUserRole(db, being_added_by, channel_id)
	if user_role >= being_added_by_role {
		return fmt.Errorf("User %s doesn't have permission to add user with role", being_added_by)
	}

	_, err := db.Exec("INSERT INTO users VALUES ($1, $2, $3) ON CONFLICT (username, channel_id) DO UPDATE SET role = $2", username, user_role, channel_id)

	if err != nil {
		return err
	}

	log.Printf("Added user %s with role %d to channel %d by %s", username, user_role, channel_id, being_added_by)
	return nil
}

func GetUsersInChannel(db *sql.DB, channel_id int) []UserWithRole {
	users := make([]UserWithRole, 0, 100)

	rows, err := db.Query("SELECT username, role FROM users WHERE channel = $1", channel_id)
	Check(err)

	for rows.Next() {
		var username string
		var role int
		rows.Scan(&username, &role)
		users = append(users, UserWithRole{username: username, role: role})
	}

	return users

}

func GetAllUsers(db *sql.DB) []UserWithRoleInChannel {
	rows, err := db.Query("SELECT username, role, channel_id FROM users")
	Check(err)

	users := make([]UserWithRoleInChannel, 0, 512)
	for rows.Next() {
		var username string
		var role int
		var channel_id int
		Check(rows.Scan(&username, &role, &channel_id))
		users = append(users, UserWithRoleInChannel{username: username, role: role, channel_id: channel_id})
	}

	rows, err = db.Query("SELECT username, role FROM priviliged_users")
	Check(err)

	for rows.Next() {
		var username string
		var role int
		Check(rows.Scan(&username, &role))
		users = append(users, UserWithRoleInChannel{username: username, role: role, channel_id: -1})
	}

	return users
}
