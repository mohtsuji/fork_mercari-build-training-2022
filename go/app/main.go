package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ImgDir = "image"
)

type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Image    string `json:"image"`
}

type Category struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type ItemData struct {
	Items []Item `json:"items"`
}

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

type MyStr string

func (s MyStr) isEmpty() bool {
	return s == ""
}

var DbConnection *sql.DB

func ErrLackItem(field string, err error, c echo.Context) error {
	c.Logger().Warnf(`Failed to create database: %v`, err) //Output Warn log for developers
	message := fmt.Sprintf(`Failed to create item. Please fill %s.`, field)
	res := Response{Message: message}
	return c.JSON(http.StatusBadRequest, res) //Response for user
}

func FailCreateItem(err error, c echo.Context) error {
	c.Logger().Errorf(`Failed to create "items.json": %v`, err) //Output Warn log for developers
	res := Response{Message: `Failed to create item`}
	return c.JSON(http.StatusInternalServerError, res) //Response for user
}

func FailGetItem(err error, c echo.Context) error {
	c.Logger().Errorf(`Failed to get item: %v`, err) //Output Warn log for developers
	res := Response{Message: `Failed to get item`}
	return c.JSON(http.StatusInternalServerError, res) //Response for user
}

func FailCreateDatabase(err error, c echo.Context) error {
	c.Logger().Errorf(`Failed to Create Database: %v`, err) //Output Warn log for developers
	res := Response{Message: `Failed to create item`}
	return c.JSON(http.StatusInternalServerError, res) //Response for user
}

func FailCreateImage(err error, c echo.Context) error {
	c.Logger().Errorf(`Failed to Create Imgae: %v`, err) //Output Warn log for developers
	res := Response{Message: `Failed to create image`}
	return c.JSON(http.StatusInternalServerError, res) //Response for user
}

func CheckLackItem(name, category MyStr, c echo.Context) (string, error) {
	if name.isEmpty() == true {
		return "name", errors.New("Name is lack")
	}
	c.Logger().Infof("Receive item: %s", name)
	if category.isEmpty() == true {
		return "category", errors.New("Category is lack")
	}
	c.Logger().Infof("Receive item: %s", category)
	return "", nil
}

func readItemJSON(c echo.Context) (ItemData, error) {
	save_items := ItemData{}
	read_items_byte, err := os.ReadFile("items.json")
	if err != nil {
		return save_items, err
	}
	if len(read_items_byte) != 0 {
		err = json.Unmarshal(read_items_byte, &save_items)
		if err != nil {
			return save_items, err
		}
	}
	return save_items, nil
}

func openDatabase() (db *sql.DB, err error) {
	//Connect database. If not exist "mercari.sql", create it.
	DbConnection, err := sql.Open("sqlite3", "./db/mercari.sqlite3")
	if err != nil {
		return nil, err
	}
	//Create table
	data, err := os.ReadFile("./db/items.db")
	if err != nil {
		return nil, err
	}
	sqlCmd := strings.Split(string(data), "\n")
	for _, cmd := range sqlCmd {
		_, err = DbConnection.Exec(cmd)
		if err != nil {
			return nil, err
		}
	}
	return DbConnection, nil
}

func UploadImage(c echo.Context) (string, error) {
	//Create images directory
	err := os.Mkdir("images", 0750) //これ0750で大丈夫？
	if err != nil && !os.IsExist(err) {
		return "", FailCreateDatabase(err, c)
	}
	//Upload file
	image, err := c.FormFile("image") //Content_typeの勉強したほうが良さそう
	//ここはエラーの種類で処理方法変えるべき？
	if err != nil {
		return "", err
		//	return "", nil
	}
	//src -> image file data
	src, err := image.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	//Get image file name
	fmt.Println(image.Filename)
	fileModel := strings.Split(image.Filename, ".")
	fileName := fileModel[0]
	//Exchange filename from string to hash256
	fileNameHash := sha256.Sum256([]byte(fileName))
	fileNameHashString := hex.EncodeToString(fileNameHash[:]) //ハッシュ値はasciiコード化された文字でもなんでもなく，そのままでstringとして扱いたい
	//Create ./images/image.jpg
	dst, err := os.Create(fmt.Sprintf("./images/%s.jpg", fileNameHashString))
	if err != nil {
		return "", err
	}
	defer dst.Close()
	//Copy image file data
	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}
	c.Logger().Infof("Receive item: %s", image.Filename)
	return fmt.Sprint(fileNameHashString, ".jpg"), nil
}

func addItem(c echo.Context) error {
	// Get form data
	name := MyStr(c.FormValue("name")) //Create Form Value
	category := MyStr(c.FormValue("category"))
	//If "name" or "category" is lack, it is error
	if s, err := CheckLackItem(name, category, c); err != nil {
		return ErrLackItem(s, err, c)
	}
	//Upload image
	image, err := UploadImage(c)
	if err != nil {
		return FailCreateImage(err, c)
	}
	//Connect database. If not exist "mercari.sql", create it.
	DbConnection, err := openDatabase()
	if err != nil {
		return FailCreateDatabase(err, c)
	}
	defer DbConnection.Close()
	//Insert items
	_, err = DbConnection.Exec("insert into category(name) values(?)", string(category))
	cmd := fmt.Sprintf(`SELECT * FROM category WHERE name='%s'`, string(category))
	category_data, err := sendCategorySelectQuery(DbConnection, cmd)
	if err != nil {
		return FailGetItem(err, c)
	}
	_, err = DbConnection.Exec("insert into items(name, category_id, image) values(?, ?, ?)", string(name), category_data.Id, image)
	if err != nil {
		return FailCreateDatabase(err, c)
	}
	message := fmt.Sprintf("item received: %s", name)
	res := Response{Message: message}
	return c.JSON(http.StatusOK, res) //Response for user
}

func sendCategorySelectQuery(DbConnection *sql.DB, query string) (Category, error) {
	// get records
	var (
		id       int
		name     string
		category Category
	)
	rows, err := DbConnection.Query(query)
	if err != nil {
		return category, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&id, &name); err != nil {
			return category, err
		}
		category = Category{Id: id, Name: name}
	}
	return category, nil
}

func sendItemsSelectQuery(DbConnection *sql.DB, query string) (ItemData, error) {
	// get records
	var (
		id             int
		name           string
		category_id    int
		image          string
		idAtCategory   int
		nameAtCategory string
		items          ItemData
	)
	rows, err := DbConnection.Query(query)
	if err != nil {
		return items, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&id, &name, &category_id, &image, &idAtCategory, &nameAtCategory); err != nil {
			return items, err
		}
		items.Items = append(items.Items, Item{Name: name, Category: nameAtCategory, Image: image})
	}
	return items, nil
}

func getItem(c echo.Context) error {
	//Connect database. If not exist "mercari.sql", create it.
	DbConnection, err := openDatabase()
	if err != nil {
		return FailCreateDatabase(err, c)
	}
	defer DbConnection.Close()
	// get records
	items, err := sendItemsSelectQuery(DbConnection, `SELECT * FROM items INNER JOIN category ON items.category_id = category.id`)
	if err != nil {
		return FailGetItem(err, c)
	}
	return c.JSONPretty(http.StatusOK, items, " ") //Go言語仕様のままでも勝手にJSONにエンコーディングしてくれる
}

func searchItem(c echo.Context) error {
	//Connect database. If not exist "mercari.sql", create it.
	DbConnection, err := openDatabase()
	if err != nil {
		return FailCreateDatabase(err, c)
	}
	defer DbConnection.Close()
	// get records
	name := c.QueryParam("keyword")
	cmd := fmt.Sprintf(`SELECT * FROM items INNER JOIN category ON items.category_id = category.id WHERE items.name='%s'`, name)
	items, err := sendItemsSelectQuery(DbConnection, cmd)
	if err != nil {
		return FailGetItem(err, c)
	}
	return c.JSONPretty(http.StatusOK, items, " ") //Go言語仕様のままでも勝手にJSONにエンコーディングしてくれる
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("itemImg"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func getDetails(c echo.Context) error {
	//Connect database. If not exist "mercari.sql", create it.
	DbConnection, err := openDatabase()
	if err != nil {
		return FailCreateDatabase(err, c)
	}
	defer DbConnection.Close()
	// get id
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return FailGetItem(err, c)
	}
	cmd := fmt.Sprintf(`SELECT * FROM items INNER JOIN category ON items.category_id = category.id WHERE items.id='%d'`, id)
	items, err := sendItemsSelectQuery(DbConnection, cmd)
	if err != nil {
		return FailGetItem(err, c)
	}
	return c.JSONPretty(http.StatusOK, items.Items[0], " ") //Go言語仕様のままでも勝手にJSONにエンコーディングしてくれる
}

func main() {
	//new instance
	e := echo.New()

	// Middlewareの設定
	//https://ken-aio.github.io/post/2019/02/06/golang-echo-middleware/
	e.Use(middleware.Logger())  //いわゆるアクセスログのようなリクエスト単位のログを出力
	e.Use(middleware.Recover()) //アプリケーションのどこかで予期せずにpanicを起こしてしまっても、サーバは落とさずにエラーレスポンスを返せるようにリカバリーする
	//set log level
	e.Logger.SetLevel(log.INFO) //e.LoggerはINFO以上のログじゃないと出力しない

	front_url := os.Getenv("FRONT_URL") //"FROTN_URL"は空なのでifに入る
	if front_url == "" {
		front_url = "http://localhost:3000" //このport番号って適当？
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url}, //今回はfront_urlに対して情報共有を許可しますよ，ということ
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		//使用可能はメソッドはGET,PUT,POST,DELTE
		//AllowCredentialsは初期値でfalse(cookieは使用しない）)
	}))

	// Routes ルートから誘導された関数はすべてreturn errだが，もしerr != nilだったら{"message":"Internal Server Error"}がレスポンスされるぽい
	e.GET("/", root) //"/"->root
	e.POST("/items", addItem)
	e.GET("/items", getItem)
	e.GET("/search", searchItem)
	e.GET("/items/:id", getDetails)
	e.GET("/image/:itemImg", getImg)

	// Start server. If e.Start return err, e.Logger.Fatal outputs log and Exit(1)
	e.Logger.Fatal(e.Start(":9000"))
}
