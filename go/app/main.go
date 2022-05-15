package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
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

var itemData = ItemData{Items: make([]Item, 0, 0)}

func ErrLackItem(field string, c echo.Context) error {
	c.Logger().Warn(`Failed to create "item.json": `, field, " is ", "empty.") //開発者用のエラーログの出力
	message := fmt.Sprintf(`Failed to create item. Please fill %s.`, field)
	res := Response{Message: message}         //ユーザー用にエラーをレスポンスする
	return c.JSON(http.StatusBadRequest, res) //ユーザー用にエラーをレスポンスする
}

func addItem(c echo.Context) error {
	//fileの作成
	file, err := os.Create("item.json") //fileはos.File型
	if err != nil {
		c.Logger().Error(`Failed to create "item.json": %v`, err) //開発者用のエラーログの出力
		res := Response{Message: `Failed to create item`}         //ユーザー用にエラーをレスポンスする
		return c.JSON(http.StatusInternalServerError, res)        //ユーザー用にエラーをレスポンスする
	}
	defer file.Close()

	// Get form data
	name := c.FormValue("name") //多分 -d name=jacketとしたときには，新たにjacketという名前のフォーム（カテゴリ？）を作成している？
	category := c.FormValue("category")
	//nameとcategoryいずれかが不足してたらエラー
	if name == "" {
		return ErrLackItem("name", c)
	} else if category == "" {
		return ErrLackItem("category", c)
	}
	c.Logger().Infof("Receive item: %s", name)
	c.Logger().Infof("Receive item: %s", category)
	itemData.Items = append(itemData.Items, Item{Name: name, Category: category})

	//サーバーがレスポンスするメッセージを指定
	message := fmt.Sprintf("item received: name=%s, category=%s", name, category)
	res := Response{Message: message}

	err = json.NewEncoder(file).Encode(itemData) //Go→JSONに変換してfileに書き込む
	if err != nil {
		c.Logger().Error(`Failed to create "item.json": %v`, err) //開発者用のエラーログの出力
		res := Response{Message: `Failed to create item`}
		return c.JSON(http.StatusInternalServerError, res) //ユーザー用にエラーをレスポンスする
	}

	return c.JSON(http.StatusOK, res) //ユーザーにレスポンスを返す。
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
	//ログの出力レベルを設定
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
	// 各ルーティングに対するハンドラを設定
	e.GET("/", root)          //"/"がきたらrootをよぶ
	e.POST("/items", addItem) //addItemでエラーが起きた場合ってどこで拾えばいいの？
	e.GET("/image/:itemImg", getImg)

	// Start server ：　Logger.Fatalは恐らくecho.Startがerrをreturnしたときに，errの内容を出力してexitする
	e.Logger.Fatal(e.Start(":9000"))
}
