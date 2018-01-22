package main

import (
  "github.com/kataras/iris"

  "github.com/kataras/iris/middleware/logger"
  "github.com/kataras/iris/middleware/recover"

  "encoding/json"
  // "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"
  "time"
)

type Student struct {
	ChannelId string `form:"channel_id"`
  ChannelName string `form:"channel_name"`
  Command string `form:"command"`
  ResponseUrl string `form:"response_url"`
  TeamDomain string `form:"team_domain"`
  TeamId string `form:"team_id"`
  Text string `form:"text"`
  Token string `form:"token"`
  TriggerId string `form:"trigger_id"`
  UserId string `form:"user_id"`
  UserName string `form:"user_name"`
}

type Response struct {
	Payload struct {
    Last string `json:"last"`
  }
}

type priceResponse struct {

}

// type Cryptocurrency struct {
// 	Last int `json:"last"`
// }

var (
  coin string
  coinText = ""
  coinColor = ""
)

func main() {
  app := iris.New()
  app.Logger().SetLevel("debug")
  app.Use(recover.New())
	app.Use(logger.New())

  app.Post("/prices", func(ctx iris.Context) {
		student := Student{}
    err := ctx.ReadForm(&student)

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
		}

    coin := strings.ToLower(student.Text)

    if (len(coin) > 0) && (coin == "btc" || coin == "eth" || coin == "xrp") {

      if coin == "btc" {
        coinText = "Bitcoin"
        coinColor = "#F2A900"
      } else if coin == "eth" {
        coinText = "Ethereum"
        coinColor = "#3C3C3D"
      } else if coin == "xrp" {
        coinText = "Ripple"
        coinColor = "#0091CF"
      } else {
        coinText = "Unknown coin"
        coinColor = "#A5C88F"
      }

      response, err := http.Get("https://api.bitso.com/v3/ticker/?book=" + coin +"_mxn")

      if err != nil {
        // fmt.Print(err.Error())
        ctx.JSON(iris.Map{
          "text": "The Bitso API is having some issues, try again in a few moments.",
          "response_type": "ephemeral",
        });
        os.Exit(1)
      }

      body, err := ioutil.ReadAll(response.Body)
      // text := string(body)

      if err != nil {
        log.Fatal(err)
      }

      cryptocurrencyData := Response{}
      json.Unmarshal(body, &cryptocurrencyData)

      if err != nil {
        log.Fatal(err)
      }


      ctx.JSON(iris.Map{
        "text": "*" + strings.ToUpper(coinText) + "* information from Bitso",
        "response_type": "in_channel",
        "mrkdwn": true,
        "attachments":[]iris.Map{iris.Map{
          "title": "Start trading!",
          "title_link": "https://bitso.com/trade",
          "text": "Last price: " + cryptocurrencyData.Payload.Last + " MXN",
          "color": coinColor,
          "ts": time.Now(),
        }},
      })
    } else {
      response, err := http.Get("https://api.bitso.com/v3/ticker")

      if err != nil {
        ctx.JSON(iris.Map{
          "text": "The Bitso API is having some issues, try again in a few moments.",
          "response_type": "ephemeral",
        });
        os.Exit(1)
      }

      body, err := ioutil.ReadAll(response.Body)
      // text := string(body)

      if err != nil {
        log.Fatal(err)
      }

      cryptocurrencyData := Response{}
      json.Unmarshal(body, &cryptocurrencyData)

      if err != nil {
        log.Fatal(err)
      }


      ctx.JSON(iris.Map{"text": "The last price of BTC, ETH and XRP is $ " + cryptocurrencyData.Payload.Last + " MXN", "response_type": "in_channel"})
    }
	})

	app.Run(iris.Addr(":8080"))
}
