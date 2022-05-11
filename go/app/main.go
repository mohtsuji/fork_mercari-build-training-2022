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

func addItem(c echo.Context) error {
	//fileの作成
	file, errf := os.Create("item.json") //fileはos.File型
	if errf != nil {
		fmt.Println(errf)
	}

	// Get form data
	itemData.Items = append(itemData.Items, Item{Name: c.FormValue("name"), Category: c.FormValue("category")})

	name := c.FormValue("name") //多分 -d name=jacketとしたときには，新たにjacketという名前のフォーム（カテゴリ？）を作成している？
	category := c.FormValue("category")
	c.Logger().Infof("Receive item: %s", name)
	c.Logger().Infof("Receive item: %s", category)

	message := fmt.Sprintf("item received: name=%s, category=%s", name, category) //これは画面に表示しているだけ
	res := Response{Message: message}                                             //サーバーがレスポンスするメッセージを指定

	err := json.NewEncoder(file).Encode(itemData)
	if err != nil {
		fmt.Println(err)
	}

	return c.JSON(http.StatusOK, res) // レスポンスをJSONに変換して表示？
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
	e.Logger.SetLevel(log.INFO) //INFOはlog level 2らしいが，具体的にはわからない・・・

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
	e.GET("/", root) //"/"がきたらrootをよぶ
	e.POST("/items", addItem)
	e.GET("/image/:itemImg", getImg)

	// Start server ：　Logger.Fatalは恐らくecho.Startがerrをreturnしたときに，errの内容を出力してexitする
	e.Logger.Fatal(e.Start(":9000")) //Loggerはinterfaseだった。多分このinterfaceの実装はecho.goがimportしている"github.com/labstack/gommon/log"の中で実装されている。しかしこの中でもさらにglobalとかいう変数を使っててもうわけわかめ
}
