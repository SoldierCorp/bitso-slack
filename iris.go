package main

/*
  Dependencies
*/
import (
  "github.com/kataras/iris"

  "github.com/kataras/iris/middleware/logger"
  "github.com/kataras/iris/middleware/recover"

  "encoding/json"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"
  "time"
)

/*
  Structs
*/

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

type GeneralResponse struct {
	PayloadData []Payload `json:"payload"`
}

type Payload struct {
  Book string `json:"book"`
	Last string `json:"last"`
}

/*
  Variables
*/
var (
  coin string
  coinText string
  coinColor string
  responseText string
)

/*
  Function: main
*/
func main() {
  app := iris.New()
  app.Logger().SetLevel("debug")
  app.Use(recover.New())
	app.Use(logger.New())

  // Handling POST /prices
  app.Post("/prices", func(ctx iris.Context) {
		student := Student{}
    err := ctx.ReadForm(&student)

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
		}

    coin := strings.ToLower(student.Text)

    // If a coin is specified, return data for that coin
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
        "text": "The latest price of *" + strings.ToUpper(coinText) + " (" + strings.ToUpper(coin) + ") * from Bitso is:",
        "response_type": "in_channel",
        "mrkdwn": true,
        "attachments":[]iris.Map{iris.Map{
          "title": "Start trading!",
          "title_link": "https://bitso.com/trade",
          "text": "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.Payload.Last + " MXN *",
          "color": coinColor,
          "ts": time.Now(),
        }},
      })

    // If a coin is not specified, return data for all of them
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

      if err != nil {
        log.Fatal(err)
      }

      cryptocurrencyData := GeneralResponse{}
      json.Unmarshal(body, &cryptocurrencyData)

      coinColor = "#A5C88F"

      for i := range cryptocurrencyData.PayloadData {
        if cryptocurrencyData.PayloadData[i].Book == "btc_mxn" {
          coinText = "Bitcoin"
          responseText = "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.PayloadData[i].Last + " MXN \n"
        } else if cryptocurrencyData.PayloadData[i].Book == "eth_mxn" {
          coinText = "Ethereum"
          responseText += "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.PayloadData[i].Last + " MXN \n"
        } else if cryptocurrencyData.PayloadData[i].Book == "xrp_mxn" {
          coinText = "Ripple"
          responseText += "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.PayloadData[i].Last + " MXN"
        }
      }

      if err != nil {
        log.Fatal(err)
      }

      ctx.JSON(iris.Map{
        "text": "The latest prices of BTC, ETH and XRP from Bitso are:",
        "response_type": "in_channel",
        "mrkdwn": true,
        "attachments":[]iris.Map{
          iris.Map{
            "title": "Start trading!",
            "title_link": "https://bitso.com/trade",
            "text": responseText,
            "color": coinColor,
            "ts": time.Now(),
          },
        },
      })
    }
	})

	app.Run(iris.Addr(":3333"))
}
