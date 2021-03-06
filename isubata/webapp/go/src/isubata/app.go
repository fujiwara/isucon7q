package main

import (
	crand "crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
)

const (
	avatarMaxBytes = 1 * 1024 * 1024
)

var (
	db            *sqlx.DB
	ErrBadReqeust = echo.NewHTTPError(http.StatusBadRequest)
)

type Renderer struct {
	templates *template.Template
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

func init() {
	seedBuf := make([]byte, 8)
	crand.Read(seedBuf)
	rand.Seed(int64(binary.LittleEndian.Uint64(seedBuf)))

	db_host := os.Getenv("ISUBATA_DB_HOST")
	if db_host == "" {
		db_host = "127.0.0.1"
	}
	db_port := os.Getenv("ISUBATA_DB_PORT")
	if db_port == "" {
		db_port = "3306"
	}
	db_user := os.Getenv("ISUBATA_DB_USER")
	if db_user == "" {
		db_user = "root"
	}
	db_password := os.Getenv("ISUBATA_DB_PASSWORD")
	if db_password != "" {
		db_password = ":" + db_password
	}

	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/isubata?parseTime=true&loc=Local&charset=utf8mb4&interpolateParams=true",
		db_user, db_password, db_host, db_port)

	log.Printf("Connecting to db: %q", dsn)
	db, _ = sqlx.Connect("mysql", dsn)
	for {
		err := db.Ping()
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(time.Second * 3)
	}

	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Printf("Succeeded to connect db.")
}

type User struct {
	ID          int64     `json:"-" db:"id"`
	Name        string    `json:"name" db:"name"`
	Salt        string    `json:"-" db:"salt"`
	Password    string    `json:"-" db:"password"`
	DisplayName string    `json:"display_name" db:"display_name"`
	AvatarIcon  string    `json:"avatar_icon" db:"avatar_icon"`
	CreatedAt   time.Time `json:"-" db:"created_at"`
}

func addMessage(channelID, userID int64, content string) (int64, error) {
	res, err := db.Exec(
		"INSERT INTO message (channel_id, user_id, content, created_at) VALUES (?, ?, ?, NOW())",
		channelID, userID, content)
	if err != nil {
		return 0, err
	}
	resId, resErr := res.LastInsertId()

	res, err = db.Exec(
		"UPDATE channel SET message_count = message_count + 1 WHERE id = ?",
		channelID,
	)
	if err != nil {
		return 0, err
	}

	return resId, resErr
}

type Message struct {
	ID              int64     `db:"id"`
	ChannelID       int64     `db:"channel_id"`
	UserID          int64     `db:"user_id"`
	Content         string    `db:"content"`
	CreatedAt       time.Time `db:"created_at"`
	UserName        string    `db:"user_name"`
	UserDisplayName string    `db:"user_display_name"`
	UserAvatarIcon  string    `db:"user_avatar_icon"`
}

func queryMessages(chanID, lastID int64) ([]Message, error) {
	msgs := []Message{}
	err := db.Select(&msgs, "SELECT m.*, u.name as user_name, u.display_name as user_display_name, u.avatar_icon as user_avatar_icon "+
		"FROM message m JOIN user u ON m.user_id = u.id WHERE m.id > ? AND m.channel_id = ? "+
		"ORDER BY m.id DESC LIMIT 100",
		lastID, chanID)
	return msgs, err
}

func sessUserID(c echo.Context) int64 {
	return sessGetInt64(c, "user_id")
}

func sessGetInt64(c echo.Context, name string) int64 {
	sess, _ := session.Get("session", c)
	var v int64
	if x, ok := sess.Values[name]; ok {
		v, _ = x.(int64)
	}
	return v
}

func sessGetString(c echo.Context, name string) string {
	sess, _ := session.Get("session", c)
	var v string
	if x, ok := sess.Values[name]; ok {
		v, _ = x.(string)
	}
	return v
}

func sessGetTime(c echo.Context, name string) time.Time {
	sess, _ := session.Get("session", c)
	var s string
	if x, ok := sess.Values[name]; ok {
		s, _ = x.(string)
	}
	t, _ := time.Parse("2006-01-02 15:04:05", s)
	return t
}

func sessSetUser(c echo.Context, user *User) {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		HttpOnly: true,
		MaxAge:   360000,
	}
	sess.Values["user_id"] = user.ID
	sess.Values["user_name"] = user.Name
	sess.Values["user_display_name"] = user.DisplayName
	sess.Values["user_avator_icon"] = user.AvatarIcon
	sess.Values["user_created_at"] = user.CreatedAt.Format("2006-01-02 15:04:05")
	sess.Save(c.Request(), c.Response())
}

func ensureLogin(c echo.Context) (*User, error) {
	userID := sessUserID(c)
	if userID == 0 {
		c.Redirect(http.StatusSeeOther, "/login")
		return nil, nil
	}

	return &User{
		ID:          userID,
		Name:        sessGetString(c, "user_name"),
		DisplayName: sessGetString(c, "user_display_name"),
		AvatarIcon:  sessGetString(c, "user_avator_icon"),
		CreatedAt:   sessGetTime(c, "user_created_at"),
	}, nil
}

const LettersAndDigits = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
	b := make([]byte, n)
	z := len(LettersAndDigits)

	for i := 0; i < n; i++ {
		b[i] = LettersAndDigits[rand.Intn(z)]
	}
	return string(b)
}

func register(name, password string) (*User, error) {
	salt := randomString(20)
	digest := fmt.Sprintf("%x", sha1.Sum([]byte(salt+password)))
	now := time.Now()

	res, err := db.Exec(
		"INSERT INTO user (name, salt, password, display_name, avatar_icon, created_at)"+
			" VALUES (?, ?, ?, ?, ?, ?)",
		name, salt, digest, name, "default.png", now.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:          id,
		Name:        name,
		Salt:        salt,
		Password:    digest,
		DisplayName: name,
		AvatarIcon:  "default.png",
		CreatedAt:   now,
	}, nil
}

// request handlers

func getInitialize(c echo.Context) error {
	db.MustExec("DELETE FROM user WHERE id > 1000")
	db.MustExec("DELETE FROM image WHERE id > 1001")
	db.MustExec("DELETE FROM channel WHERE id > 10")
	db.MustExec("DELETE FROM message WHERE id > 10000")
	db.MustExec("DELETE FROM haveread")
	db.MustExec("UPDATE channel JOIN (SELECT channel_id, COUNT(*) msg_count FROM message GROUP BY channel_id) tmp ON tmp.channel_id = channel.id SET message_count = msg_count")
	return c.String(204, "")
}

func getIndex(c echo.Context) error {
	userID := sessUserID(c)
	if userID != 0 {
		return c.Redirect(http.StatusSeeOther, "/channel/1")
	}

	return c.Render(http.StatusOK, "index", map[string]interface{}{
		"ChannelID": nil,
	})
}

type ChannelInfo struct {
	ID           int64     `db:"id"`
	Name         string    `db:"name"`
	Description  string    `db:"description"`
	UpdatedAt    time.Time `db:"updated_at"`
	CreatedAt    time.Time `db:"created_at"`
	MessageCount int       `db:"message_count"`
}

func getChannel(c echo.Context) error {
	user, err := ensureLogin(c)
	if user == nil {
		return err
	}
	cID, err := strconv.Atoi(c.Param("channel_id"))
	if err != nil {
		return err
	}
	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}

	var desc string
	for _, ch := range channels {
		if ch.ID == int64(cID) {
			desc = ch.Description
			break
		}
	}
	return c.Render(http.StatusOK, "channel", map[string]interface{}{
		"ChannelID":   cID,
		"Channels":    channels,
		"User":        user,
		"Description": desc,
	})
}

func getRegister(c echo.Context) error {
	return c.Render(http.StatusOK, "register", map[string]interface{}{
		"ChannelID": 0,
		"Channels":  []ChannelInfo{},
		"User":      nil,
	})
}

func postRegister(c echo.Context) error {
	name := c.FormValue("name")
	pw := c.FormValue("password")
	if name == "" || pw == "" {
		return ErrBadReqeust
	}
	user, err := register(name, pw)
	if err != nil {
		if merr, ok := err.(*mysql.MySQLError); ok {
			if merr.Number == 1062 { // Duplicate entry xxxx for key zzzz
				return c.NoContent(http.StatusConflict)
			}
		}
		return err
	}
	sessSetUser(c, user)
	return c.Redirect(http.StatusSeeOther, "/")
}

func getLogin(c echo.Context) error {
	return c.Render(http.StatusOK, "login", map[string]interface{}{
		"ChannelID": 0,
		"Channels":  []ChannelInfo{},
		"User":      nil,
	})
}

func postLogin(c echo.Context) error {
	name := c.FormValue("name")
	pw := c.FormValue("password")
	if name == "" || pw == "" {
		return ErrBadReqeust
	}

	var user User
	err := db.Get(&user, "SELECT * FROM user WHERE name = ?", name)
	if err == sql.ErrNoRows {
		return echo.ErrForbidden
	} else if err != nil {
		return err
	}

	digest := fmt.Sprintf("%x", sha1.Sum([]byte(user.Salt+pw)))
	if digest != user.Password {
		return echo.ErrForbidden
	}
	sessSetUser(c, &user)
	return c.Redirect(http.StatusSeeOther, "/")
}

func getLogout(c echo.Context) error {
	sess, _ := session.Get("session", c)
	delete(sess.Values, "user_id")
	sess.Save(c.Request(), c.Response())
	return c.Redirect(http.StatusSeeOther, "/")
}

func postMessage(c echo.Context) error {
	user, err := ensureLogin(c)
	if user == nil {
		return err
	}

	message := c.FormValue("message")
	if message == "" {
		return echo.ErrForbidden
	}

	var chanID int64
	if x, err := strconv.Atoi(c.FormValue("channel_id")); err != nil {
		return echo.ErrForbidden
	} else {
		chanID = int64(x)
	}

	if _, err := addMessage(chanID, user.ID, message); err != nil {
		return err
	}

	return c.NoContent(204)
}

func getMessage(c echo.Context) error {
	userID := sessUserID(c)
	if userID == 0 {
		return c.NoContent(http.StatusForbidden)
	}

	chanID, err := strconv.ParseInt(c.QueryParam("channel_id"), 10, 64)
	if err != nil {
		return err
	}
	lastID, err := strconv.ParseInt(c.QueryParam("last_message_id"), 10, 64)
	if err != nil {
		return err
	}

	messages, err := queryMessages(chanID, lastID)
	if err != nil {
		return err
	}

	response := make([]map[string]interface{}, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		r := make(map[string]interface{})
		r["id"] = m.ID
		r["user"] = User{
			ID:          m.UserID,
			Name:        m.UserName,
			DisplayName: m.UserDisplayName,
			AvatarIcon:  m.UserAvatarIcon,
		}
		r["date"] = m.CreatedAt.Format("2006/01/02 15:04:05")
		r["content"] = m.Content
		response = append(response, r)
	}

	if len(messages) > 0 {
		_, err := db.Exec("INSERT INTO haveread (user_id, channel_id, message_id, updated_at, created_at)"+
			" VALUES (?, ?, ?, NOW(), NOW())"+
			" ON DUPLICATE KEY UPDATE message_id = ?, updated_at = NOW()",
			userID, chanID, messages[0].ID, messages[0].ID)
		if err != nil {
			return err
		}
	}

	return c.JSON(http.StatusOK, response)
}

func queryChannels() ([]int64, error) {
	res := []int64{}
	err := db.Select(&res, "SELECT id FROM channel")
	return res, err
}

func fetchUnread(c echo.Context) error {
	userID := sessUserID(c)
	if userID == 0 {
		return c.NoContent(http.StatusForbidden)
	}

	time.Sleep(time.Second)

	channels, err := queryChannels()
	if err != nil {
		return err
	}

	resp := []map[string]interface{}{}

	type HaveRead struct {
		ChannelID int64 `db:"channel_id"`
		MessageID int64 `db:"message_id"`
	}

	// lastIDをまとめて取ってくる
	haveReads := make(map[int64]int64)
	{
		query, args, err := sqlx.In(
			"SELECT channel_id, message_id FROM haveread WHERE user_id = ? AND channel_id in (?)",
			userID, channels)
		if err != nil {
			return err
		}

		hs := []HaveRead{}
		err = db.Select(&hs, query, args...)
		if err != nil {
			return err
		}

		for _, h := range hs {
			haveReads[h.ChannelID] = h.MessageID
		}
	}

	noReadChanIDs := []int64{}

	for _, chID := range channels {
		lastID := haveReads[chID]

		// 未読情報がないものはあとでまとめて取る
		if lastID == 0 {
			noReadChanIDs = append(noReadChanIDs, chID)
			continue
		}

		var cnt int64
		err = db.Get(&cnt,
			"SELECT COUNT(*) as cnt FROM message WHERE channel_id = ? AND ? < id",
			chID, lastID)
		if err != nil {
			return err
		}
		r := map[string]interface{}{
			"channel_id": chID,
			"unread":     cnt}
		resp = append(resp, r)
	}

	// 未読情報がないもの
	if 0 < len(noReadChanIDs) {
		query, args, err := sqlx.In("SELECT * FROM channel WHERE id in (?)",
			noReadChanIDs)
		if err != nil {
			return nil
		}

		chans := []ChannelInfo{}
		err = db.Select(&chans, query, args...)
		if err != nil {
			return err
		}

		for _, ch := range chans {
			r := map[string]interface{}{
				"channel_id": ch.ID,
				"unread":     ch.MessageCount}
			resp = append(resp, r)
		}
	}

	return c.JSON(http.StatusOK, resp)
}

func getHistory(c echo.Context) error {
	chID, err := strconv.ParseInt(c.Param("channel_id"), 10, 64)
	if err != nil || chID <= 0 {
		return ErrBadReqeust
	}

	user, err := ensureLogin(c)
	if user == nil {
		return err
	}

	var page int64
	pageStr := c.QueryParam("page")
	if pageStr == "" {
		page = 1
	} else {
		page, err = strconv.ParseInt(pageStr, 10, 64)
		if err != nil || page < 1 {
			return ErrBadReqeust
		}
	}

	const N = 20
	var cnt int64
	err = db.Get(&cnt, "SELECT message_count as cnt FROM channel WHERE id = ?", chID)
	if err != nil {
		return err
	}
	maxPage := int64(cnt+N-1) / N
	if maxPage == 0 {
		maxPage = 1
	}
	if page > maxPage {
		return ErrBadReqeust
	}

	// idのみをselect
	messageIDs := []int{}
	err = db.Select(&messageIDs,
		"SELECT id FROM message WHERE channel_id = ? ORDER BY id DESC LIMIT ? OFFSET ?",
		chID, N, (page-1)*N)
	if err != nil {
		return err
	}

	// userをjoinして内容取得
	messages := []Message{}
	if 0 < len(messageIDs) {
		query, args, err := sqlx.In("SELECT m.*, u.name as user_name, u.display_name as user_display_name, u.avatar_icon as user_avatar_icon "+
			"FROM message m JOIN user u ON m.user_id = u.id WHERE m.id in (?) ORDER BY FIELD (m.id, ?)",
			messageIDs, messageIDs)
		if err != nil {
			return err
		}

		err = db.Select(&messages, query, args...)
		if err != nil {
			return err
		}
	}

	mjson := make([]map[string]interface{}, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		// getMessageでやってることと一緒
		m := messages[i]
		r := make(map[string]interface{})
		r["id"] = m.ID
		r["user"] = User{
			ID:          m.UserID,
			Name:        m.UserName,
			DisplayName: m.UserDisplayName,
			AvatarIcon:  m.UserAvatarIcon,
		}
		r["date"] = m.CreatedAt.Format("2006/01/02 15:04:05")
		r["content"] = m.Content
		mjson = append(mjson, r)
	}

	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "history", map[string]interface{}{
		"ChannelID": chID,
		"Channels":  channels,
		"Messages":  mjson,
		"MaxPage":   maxPage,
		"Page":      page,
		"User":      user,
	})
}

func getProfile(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}

	userName := c.Param("user_name")
	var other User
	err = db.Get(&other, "SELECT * FROM user WHERE name = ?", userName)
	if err == sql.ErrNoRows {
		return echo.ErrNotFound
	}
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "profile", map[string]interface{}{
		"ChannelID":   0,
		"Channels":    channels,
		"User":        self,
		"Other":       other,
		"SelfProfile": self.ID == other.ID,
	})
}

func getAddChannel(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "add_channel", map[string]interface{}{
		"ChannelID": 0,
		"Channels":  channels,
		"User":      self,
	})
}

func postAddChannel(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	name := c.FormValue("name")
	desc := c.FormValue("description")
	if name == "" || desc == "" {
		return ErrBadReqeust
	}

	res, err := db.Exec(
		"INSERT INTO channel (name, description, updated_at, created_at) VALUES (?, ?, NOW(), NOW())",
		name, desc)
	if err != nil {
		return err
	}
	lastID, _ := res.LastInsertId()
	return c.Redirect(http.StatusSeeOther,
		fmt.Sprintf("/channel/%v", lastID))
}

func postProfile(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	avatarName := ""
	var avatarData []byte
	editName := c.FormValue("display_name")

	if fh, err := c.FormFile("avatar_icon"); err == http.ErrMissingFile {
		// no file upload
	} else if err != nil {
		return err
	} else {
		dotPos := strings.LastIndexByte(fh.Filename, '.')
		if dotPos < 0 {
			return ErrBadReqeust
		}
		ext := fh.Filename[dotPos:]
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif":
			break
		default:
			return ErrBadReqeust
		}

		file, err := fh.Open()
		if err != nil {
			return err
		}
		avatarData, _ = ioutil.ReadAll(file)
		file.Close()

		if len(avatarData) > avatarMaxBytes {
			return ErrBadReqeust
		}

		avatarName = fmt.Sprintf("%x%s", sha1.Sum(avatarData), ext)
	}

	updateAvatarIcon := ""

	if avatarName != "" && len(avatarData) > 0 {
		// 画像をローカルに保存(テンポラリファイルから保存先にリネーム)
		path := `/home/isucon/isubata/webapp/public/icons/` + imageDir + avatarName
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// ファイルがないときだけ書く
			err := ioutil.WriteFile(path, avatarData, 0644)
			if err != nil {
				return err
			}
		}

		updateAvatarIcon = imageDir + avatarName
	}

	err = nil
	if updateAvatarIcon != "" {
		if editName != "" {
			_, err = db.Exec("UPDATE user SET avatar_icon = ?, display_name = ? WHERE id = ?",
				updateAvatarIcon, editName, self.ID)

			self.AvatarIcon = updateAvatarIcon
			self.DisplayName = editName
		} else {
			_, err = db.Exec("UPDATE user SET avatar_icon = ? WHERE id = ?",
				updateAvatarIcon, self.ID)

			self.AvatarIcon = updateAvatarIcon
		}

	} else if editName != "" {
		_, err = db.Exec("UPDATE user SET display_name = ? WHERE id = ?",
			editName, self.ID)

		self.DisplayName = editName
	}
	if err != nil {
		return err
	}

	sessSetUser(c, self)

	return c.Redirect(http.StatusSeeOther, "/")
}

func getIcon(c echo.Context) error {
	// ローカルにファイルあったらそれを使う
	fileName := c.Param("file_name")
	path := `/home/isucon/isubata/webapp/public/icons/` + fileName
	if _, err := os.Stat(path); os.IsExist(err) {
		return c.File(path)
	}

	var name string
	var data []byte
	err := db.QueryRow("SELECT name, data FROM image WHERE name = ? LIMIT 1",
		fileName).Scan(&name, &data)
	if err == sql.ErrNoRows {
		return echo.ErrNotFound
	}
	if err != nil {
		return err
	}

	mime := ""
	switch true {
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		mime = "image/jpeg"
	case strings.HasSuffix(name, ".png"):
		mime = "image/png"
	case strings.HasSuffix(name, ".gif"):
		mime = "image/gif"
	default:
		return echo.ErrNotFound
	}
	return c.Blob(http.StatusOK, mime, data)
}

func tAdd(a, b int64) int64 {
	return a + b
}

func tRange(a, b int64) []int64 {
	r := make([]int64, b-a+1)
	for i := int64(0); i <= (b - a); i++ {
		r[i] = a + i
	}
	return r
}

var imageDir string

func main() {
	e := echo.New()
	funcs := template.FuncMap{
		"add":    tAdd,
		"xrange": tRange,
	}
	e.Renderer = &Renderer{
		templates: template.Must(template.New("").Funcs(funcs).ParseGlob("views/*.html")),
	}
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secretonymoris"))))
	//e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
	//	Format: "request:\"${method} ${uri}\" status:${status} latency:${latency} (${latency_human}) bytes:${bytes_out}\n",
	//}))
	e.Use(middleware.Static("../public"))

	e.GET("/initialize", getInitialize)
	e.GET("/", getIndex)
	e.GET("/register", getRegister)
	e.POST("/register", postRegister)
	e.GET("/login", getLogin)
	e.POST("/login", postLogin)
	e.GET("/logout", getLogout)

	e.GET("/channel/:channel_id", getChannel)
	e.GET("/message", getMessage)
	e.POST("/message", postMessage)
	e.GET("/fetch", fetchUnread)
	e.GET("/history/:channel_id", getHistory)

	e.GET("/profile/:user_name", getProfile)
	e.POST("/profile", postProfile)

	e.GET("add_channel", getAddChannel)
	e.POST("add_channel", postAddChannel)
	e.GET("/icons/:file_name", getIcon)

	// ホスト名からディレクトリ決める
	host, err := os.Hostname()
	if err != nil {
		panic(`failed to get hostname`)
	}
	fmt.Println("hostname = ", host)
	imageDir = fmt.Sprintf("0%s/", host[len(host)-1:])

	e.Start(":5000")
}
