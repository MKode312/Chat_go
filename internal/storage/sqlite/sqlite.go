package sqlite

import (
	"chat_go/internal/lib/api/models"
	"chat_go/internal/lib/jwts"
	"chat_go/internal/storage"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type Storage struct {
	db *sql.DB
}

type User struct {
	models.User
	password string
	id       int64
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", "./storage/chat.db")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt1, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS users(
	id INTEGER PRIMARY KEY,
	bio TEXT NOT NULL,
	password TEXT NOT NULL,
	nickname TEXT NOT NULL,
	username TEXT NOT NULL UNIQUE);
	`)

	stmt2, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS messages(
	id INTEGER PRIMARY KEY,
	sender TEXT NOT NULL,
	chatName TEXT NOT NULL,
	chatID INTEGER,
	text TEXT NOT NULL);
	`)

	stmt3, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS chats(
	id INTEGER PRIMARY KEY,
	name TEXT NOT NULL,
	messages TEXT,
	participantsUsernames TEXT NOT NULL UNIQUE,
	numberOfParticipants INTEGER);
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	_, err = stmt1.Exec()
	_, err = stmt2.Exec()
	_, err = stmt3.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(bio string, password string, nickname string, username string) (int64, error) {
	const op = "storage.sqlite.SaveUser"

	stmt, err := s.db.Prepare("INSERT INTO users(nickname, username, password, bio) VALUES(?, ?, ?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	hashedPswrd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(nickname, username, hashedPswrd, bio)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserAlreadyExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetUser(username string) (models.User, error) {
	const op = "storage.sqlite.GetChat"

	stmt, err := s.db.Prepare("SELECT bio, nickname FROM users WHERE username = ?")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var bio string
	var nickname string

	err = stmt.QueryRow(username).Scan(&bio, &nickname)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, storage.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("%s: execute statement: %w", op, err)
	}
	user := models.User{
		Username: username,
		Nickname: nickname,
		Bio:      bio,
	}

	return user, nil
}

func (s *Storage) GetNicknameByUsername(username string) (string, error) {
	const op = "storage.sqlite.GetNicknameByUsername"

	stmt, err := s.db.Prepare("SELECT nickname FROM users WHERE username = ?")
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var nickname string

	err = stmt.QueryRow(username).Scan(&nickname)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrUserNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return nickname, nil
}

func (s *Storage) DeleteUser(username string) error {
	const op = "storage.sqlite.DeleteUser"

	stmt, err := s.db.Prepare("DELETE FROM users WHERE username = ?")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	_, err = stmt.Exec(username)
	if err != nil {
		return fmt.Errorf("%s: execute statemnet %w", op, err)
	}

	return nil
}

func (s *Storage) LoginUser(username, password string) (string, error) {

	q := `
	SELECT id, password FROM users WHERE username = ?
	`
	var user User

	err := s.db.QueryRow(q, username).Scan(&user.id, &user.password)
	if err != nil {
		return "", err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.password), []byte(password))
	if err != nil {
		return "", err
	}

	token, err := jwts.GenerateJWTToken(user.id)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Storage) MakeChat(name string, usernames string) (int64, error) {
	const op = "storage.sqlite.MakeChat"

	stmt, err := s.db.Prepare("INSERT INTO chats(name, participantsUsernames) VALUES(?,	 ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(name, usernames)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetAllMessagesByChatName(chatName string) ([]models.Message, error) {

	var (
		messages []models.Message
		q        = `SELECT text FROM messages WHERE chatName = ?`
	)

	rows, err := s.db.Query(q, chatName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		msg := models.Message{}
		err := rows.Scan(&msg.Sender, &msg.Text)
		if err != nil {
			return nil, err
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *Storage) GetParticipantsByChatNameAndID(chatName string, id int64) (string, error) {
	const op = "storage.sqlite.GetParticipantsByChatNameAndID"

	stmt, err := s.db.Prepare("SELECT participantsUsernames FROM chats WHERE name = ? AND id = ?")
	if err != nil {
		return "", fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var participants string

	err = stmt.QueryRow(chatName, id).Scan(&participants)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrChatNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return participants, nil
}

func (s *Storage) SaveMessage(sender string, chatName string, chatID int64, text string) (int64, error) {
	const op = "storage.sqlite.SaveMessage"

	stmt, err := s.db.Prepare("INSERT INTO messages(sender, chatName, chatID, text) VALUES(?, ?, ?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.Exec(sender, chatName, chatID, text)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetSenderOfMessageByChatName(chatName string) (string, error) {
	const op = "storage.sqlite.GetSenderOfMessageByChatName"

	stmt, err := s.db.Prepare("SELECT sender FROM messages WHERE chatName = ?")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	var sender string

	err = stmt.QueryRow(chatName).Scan(&sender)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrMessageNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return sender, nil
}

func (s *Storage) GetAllMessagesByChatnameAndID(chatName string, id int64) ([]models.Message, error) {
	const op = "storage.sqlite.GetAllMessagesBySenderAndChatname"

	stmt, err := s.db.Prepare("SELECT text, sender, id FROM messages WHERE chatName = ? AND chatID = ?")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var messages []models.Message

	rows, err := stmt.Query(chatName, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	for rows.Next() {
		msg := models.Message{}
		err := rows.Scan(&msg.Text, &msg.Sender, &msg.ID)
		if err != nil {
			return nil, err
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
