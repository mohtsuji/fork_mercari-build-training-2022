package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
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

func ErrLackItem(field string, c echo.Context) error {
	c.Logger().Warnf(`Failed to create "items.json": `, field, " is ", "empty.") //Output Warn log for developers
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
	c.Logger().Errorf(`Failed to get "items.json": %v`, err) //Output Warn log for developers
	res := Response{Message: `Failed to get item`}
	return c.JSON(http.StatusInternalServerError, res) //Response for user
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

func addItem(c echo.Context) error {
	// Get form data
	name := MyStr(c.FormValue("name")) //Create Form Value
	category := MyStr(c.FormValue("category"))
	//If "name" or "category" is lack, it is error
	if name.isEmpty() == true {
		return ErrLackItem("name", c)
	}
	c.Logger().Infof("Receive item: %s", name)
	if category.isEmpty() == true {
		return ErrLackItem("category", c)
	}
	c.Logger().Infof("Receive item: %s", category)

	//read "items.json"
	file, err := os.OpenFile("items.json", os.O_RDWR|os.O_CREATE, 0775) //file->type os.File
	if err != nil {
		return FailCreateItem(err, c)
	}
	defer file.Close()
	save_items, err := readItemJSON(c)
	if err != nil {
		return FailCreateItem(err, c)
	}
	save_items.Items = append(save_items.Items, Item{Name: string(name), Category: string(category)})

	//encoding and write contents to "items.json"
	data := json.NewEncoder(file) //output stream = file
	data.SetIndent("", " ")       //set Indent
	err = data.Encode(save_items) //Go→JSON(encoding)
	if err != nil {
		return FailCreateItem(err, c)
	}

	//connect database
	/*	DbConnection, err := sql.Open("sqlite3", "../db/mercari.sql")
		if err != nil {
			//		return FailCreateDatabase(err, c)
		}
		defer DbConnection.Close()
		err = DbConnection.Ping()
		if err != nil {
			//	log.Fatal(err)
		}
	*/
	message := fmt.Sprintf("item received: %s", name)
	res := Response{Message: message}
	return c.JSON(http.StatusOK, res) //Response for user
}

func getItem(c echo.Context) error {
	save_items, err := readItemJSON(c)
	if err != nil {
		return FailGetItem(err, c)
	}
	return c.JSONPretty(http.StatusOK, save_items, " ") //Go言語仕様のままでも勝手にJSONにエンコーディングしてくれる
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

	// Routes
	e.GET("/", root) //"/"->root
	e.POST("/items", addItem)
	e.GET("/items", getItem)
	e.GET("/image/:itemImg", getImg)

	// Start server. If e.Start return err, e.Logger.Fatal outputs log and Exit(1)
	e.Logger.Fatal(e.Start(":9000"))
}
