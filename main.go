package main

/*
  Dependencies
*/
import (
  "database/sql"
  "errors"
	"os"
	"sort"
  "encoding/json"
  "io/ioutil"
  "log"
  "net/http"
  "strings"
  "fmt"

  "github.com/kataras/iris"

  "github.com/kataras/iris/middleware/logger"
  "github.com/kataras/iris/middleware/recover"

  "github.com/kataras/iris/sessions"
  "github.com/gorilla/securecookie"
  "github.com/markbates/goth"
  "github.com/markbates/goth/providers/slack"

  "github.com/joho/godotenv"
  _"github.com/go-sql-driver/mysql"
)

/*
  Structs
*/

type Slack struct {
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
	Payload []struct {
    Book string `json:"book"`
	  Last string `json:"last"`
  } `json:"payload"`
}

type SlackAuthAccess struct {
  AccessToken string `json:"access_token"`
  Scope string `json:"scope"`
}

type (
  SlackWorkspace struct {
    Error string `json:"error"`
    Id string `json:"id"`
    Name string `json:"name"`
    Token string `json:"token"`
  }
)

/*
  Constants
*/
const (
	DefaultTitle  = "Bitso for Slack"
	DefaultLayout = "web/layouts/master.pug"
)

/*
  Variables
*/
var (
  coin string
  coinText string
  coinColor string
  responseText string
)

var sessionsManager *sessions.Sessions


/*
  Function: init
*/
func init() {
  fmt.Println("# Init!")
	// attach a session manager
	cookieName := "bitsosessionid"
	// AES only supports key sizes of 16, 24 or 32 bytes.
	// You either need to provide exactly that amount or you derive the key from what you type in.
	hashKey := []byte("the-big-and-secret-fash-key-here")
	blockKey := []byte("lot-secret-of-characters-big-too")
	secureCookie := securecookie.New(hashKey, blockKey)

	sessionsManager = sessions.New(sessions.Config{
		Cookie: cookieName,
		Encode: secureCookie.Encode,
		Decode: secureCookie.Decode,
	})
}

/*
  These are some function helpers that you may use if you want

  GetProviderName is a function used to get the name of a provider
  for a given request. By default, this provider is fetched from
  the URL query string. If you provide it in a different way,
  assign your own function to this variable that returns the provider
  name for your request.
*/
var GetProviderName = func(ctx iris.Context) (string, error) {
  fmt.Println("# Get provider Name")

	// try to get it from the url param "provider"
	if p := ctx.URLParam("provider"); p != "" {
		return p, nil
	}

  fmt.Println("Error getting URLParam provider")
	// try to get it from the url PATH parameter "{provider} or :provider or {provider:string} or {provider:alphabetical}"
	if p := ctx.Params().Get("provider"); p != "" {
		return p, nil
	}

  fmt.Println("Error getting Params provider")


	// try to get it from context's per-request storage
	if p := ctx.Values().GetString("provider"); p != "" {
		return p, nil
  }

  fmt.Println("Error getting Values Getstring provider")

	// if not found then return an empty string with the corresponding error
	return "", errors.New("you must select a provider")
}

/*
  BeginAuthHandler is a convenience handler for starting the authentication process.
  It expects to be able to get the name of the provider from the query parameters
  as either "provider" or ":provider".

  BeginAuthHandler will redirect the user to the appropriate authentication end-point
  for the requested provider.

  See https://github.com/markbates/goth/examples/main.go to see this in action.
*/
func BeginAuthHandler(ctx iris.Context) {
  fmt.Println("# Function BeginAuthHandler")

  url, err := GetAuthURL(ctx)

	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		ctx.Writef("%v", err)
		return
	}

	ctx.Redirect(url, iris.StatusTemporaryRedirect)
}

/*
  GetAuthURL starts the authentication process with the requested provided.
  It will return a URL that should be used to send users to.

  It expects to be able to get the name of the provider from the query parameters
  as either "provider" or ":provider" or from the context's value of "provider" key.

  I would recommend using the BeginAuthHandler instead of doing all of these steps
  yourself, but that's entirely up to you.
*/
func GetAuthURL(ctx iris.Context) (string, error) {
  fmt.Println("# Function GetAuthURL")
	providerName, err := GetProviderName(ctx)
	if err != nil {
		return "", err
	}
  fmt.Println()
	provider, err := goth.GetProvider(providerName)
	if err != nil {
		return "", err
	}
	sess, err := provider.BeginAuth(SetState(ctx))
	if err != nil {
		return "", err
	}

	url, err := sess.GetAuthURL()
	if err != nil {
		return "", err
  }

	session := sessionsManager.Start(ctx)
  session.Set(providerName, sess.Marshal())

  fmt.Println("Session ")
  fmt.Println(session)

	return url, nil
}

/*
  SetState sets the state string associated with the given request.
  If no state string is associated with the request, one will be generated.
  This state is sent to the provider and can be retrieved during the
  callback.
*/
var SetState = func(ctx iris.Context) string {
	state := ctx.URLParam("state")
	if len(state) > 0 {
		return state
	}

	return "state"

}

/*
  GetState gets the state returned by the provider during the callback.
  This is used to prevent CSRF attacks, see
  http://tools.ietf.org/html/rfc6749#section-10.12
*/
var GetState = func(ctx iris.Context) string {
	return ctx.URLParam("state")
}

/*
  CompleteUserAuth does what it says on the tin. It completes the authentication
  process and fetches all of the basic information about the user from the provider.

  It expects to be able to get the name of the provider from the query parameters
  as either "provider" or ":provider".

  See https://github.com/markbates/goth/examples/main.go to see this in action.
*/
var CompleteUserAuth = func(ctx iris.Context) (goth.User, error) {
	providerName, err := GetProviderName(ctx)
	if err != nil {
		return goth.User{}, err
	}

	provider, err := goth.GetProvider(providerName)
	if err != nil {
		return goth.User{}, err
	}
	session := sessionsManager.Start(ctx)
	value := session.GetString(providerName)
	if value == "" {
		return goth.User{}, errors.New("session value for " + providerName + " not found")
	}

	sess, err := provider.UnmarshalSession(value)
	if err != nil {
		return goth.User{}, err
	}

	user, err := provider.FetchUser(sess)
	if err == nil {
		// user can be found with existing session data
		return user, err
	}

	// get new token and retry fetch
	_, err = sess.Authorize(provider, ctx.Request().URL.Query())
	if err != nil {
		return goth.User{}, err
	}

	session.Set(providerName, sess.Marshal())
	return provider.FetchUser(sess)
}

// Logout invalidates a user session.
func Logout(ctx iris.Context) error {
	providerName, err := GetProviderName(ctx)
	if err != nil {
		return err
	}
	session := sessionsManager.Start(ctx)
	session.Delete(providerName)
	return nil
}

// End of the "some function helpers".


/*
  Function: get all cryptocurrencies values
*/
func getAllCoins(ctx iris.Context) map[string]string {
  response, err := http.Get("https://api.bitso.com/v3/ticker")

  if err != nil {
    ctx.JSON(iris.Map{
      "text": "The Bitso API is having some issues, try again in a few moments.",
      "response_type": "ephemeral",
    });
    os.Exit(1)
  }

  body, err := ioutil.ReadAll(response.Body)

  cryptocurrencyData := GeneralResponse{}
  e := json.Unmarshal([]byte(string(body)), &cryptocurrencyData)

  if e != nil {
    panic(err)
  }

  var coins map[string]string
  coins = make(map[string]string)

  for i := range cryptocurrencyData.Payload {
    var current = cryptocurrencyData.Payload[i]

    if (current.Book == "btc_mxn" || current.Book == "eth_mxn" || current.Book == "xrp_mxn" || current.Book == "ltc_mxn") {
      coins[current.Book] = current.Last
    }
  }

  return coins
}

/*
  Function: main
*/
func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }

  goth.UseProviders(
    slack.New(os.Getenv("SLACK_KEY"), os.Getenv("SLACK_SECRET"), "http://localhost:3333/auth/slack/callback", "commands", "users:read"),
  )

  m := make(map[string]string)
  m["slack"] = "Slack"

  var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

  // Iris instance initialization
  app := iris.New()
  app.Logger().SetLevel("debug")
  app.Use(recover.New())
	app.Use(logger.New())

  // Static files
  app.Favicon("./web/assets/images/bitso-slack-logo.png")
  app.StaticWeb("/static", "./web/assets")

  // attach and build the templates
  tmpl := iris.Pug("./web/templates", ".pug").Reload(true)
  tmpl.Layout("layouts/master.pug").Reload(true)

  // tmpl.Binary(Asset, AssetNames)

  app.RegisterView(tmpl)


  // Handling GET /auth/slack/callback
	app.Get("/auth/{provider}/callback", func(ctx iris.Context) {
    fmt.Println("AUTH CALLBACK");

    user, err := CompleteUserAuth(ctx)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.Writef("%v", err)
			return
    }
    var c = getAllCoins(ctx)

    fmt.Println("COINS")
    fmt.Println(c)
    fmt.Println("USER")
    fmt.Println(user.AccessToken)

    // ctx.ViewData("", user)
    ctx.ViewData("Coins", c)

		if err := ctx.View("success.pug"); err != nil {
			ctx.Writef("%v", err)
    }


    // body, err := ioutil.ReadAll(user)
    // text := string(body)

    // fmt.Println(text)


    fmt.Println("USER");
    fmt.Println(user);

    // response, err := http.Get("https://api.bitso.com/v3/ticker/?book=" + coin +"_mxn")

    //   if err != nil {
    //     ctx.JSON(iris.Map{
    //       "text": "The Bitso API is having some issues, try again in a few moments.",
    //       "response_type": "ephemeral",
    //     });
    //     os.Exit(1)
    //   }

    //   body, err := ioutil.ReadAll(response.Body)
    //   // text := string(body)

    //   if err != nil {
    //     log.Fatal(err)
    //   }

    //   cryptocurrencyData := Response{}
    //   json.Unmarshal(body, &cryptocurrencyData)

    //   if err != nil {
    //     log.Fatal(err)
    //   }

		// if err != nil {
    //   ctx.Redirect("/success");
		// 	return
    // }


    // ctx.ViewData("", user)
    // ctx.Redirect("/success");
  })

  app.Get("/logout/{provider}", func(ctx iris.Context) {
		Logout(ctx)
		ctx.Redirect("/", iris.StatusTemporaryRedirect)
  })

  // Handling GET /auth/slack
  app.Get("/auth/{provider}", func(ctx iris.Context) {
    // try to get the user without re-authenticating

    // try to get the user without re-authenticating
		if gothUser, err := CompleteUserAuth(ctx); err == nil {
      fmt.Println("IF GOTHUSER")
      var c = getAllCoins(ctx)
      ctx.ViewData("Coins", c)
      ctx.ViewData("", gothUser)

			if err := ctx.View("success.pug"); err != nil {
				ctx.Writef("%v", err)
			}
		} else {
      fmt.Println("BeginAuthHandler")
			BeginAuthHandler(ctx)
    }

		// if gothUser, err := CompleteUserAuth(ctx); err == nil {

    //   fmt.Println("IF GOTHUSER")
    //   fmt.Println(gothUser)

    //   // if gothUser.NickName == "" {
    //   //   ctx.Redirect("/logout/slack");
    //   // }

    //   ctx.Redirect("/success");

    //   var c = getAllCoins(ctx);

    //   ctx.ViewData("", gothUser)
    //   ctx.ViewData("Coins", c)

    //   if err := ctx.View("index.pug"); err != nil {
		// 		ctx.Writef("%v", err)
		// 	}
		// } else {
    //   fmt.Println("BeginAuthHandler")
		// 	BeginAuthHandler(ctx)
		// }
	})

  // Handling GET /
	app.Get("/", func(ctx iris.Context) {

    var c = getAllCoins(ctx)
    ctx.ViewData("Coins", c)

		if err := ctx.View("index.pug"); err != nil {
			ctx.Writef("%v", err)
		}
  })

  // Handling GET /success
  app.Get("/success", func(ctx iris.Context) {

    var c = getAllCoins(ctx);
    ctx.ViewData("Coins", c)

		if err := ctx.View("success.pug"); err != nil {
			ctx.Writef("%v", err)
		}
  })

  // Handling POST /prices
  app.Post("/localprices", func(ctx iris.Context) {
		slack := Slack{}
    err := ctx.ReadForm(&slack)

    if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
    }

    // coin := strings.ToLower(slack.Text)

    log.Print(slack)

    db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/bitso-slack")

    if err != nil {
      fmt.Println(err.Error())
      response := SlackWorkspace{Id: "", Error: "true", Name: "", Token: ""}
      ctx.JSON(iris.Map{
        "status": 400,
        "message": response,
      })
    } else {
      fmt.Println("- DB success")
    }

    var (
      id int
      name string
      token string
    )

    rows, err:= db.Query("SELECT id, name, token FROM workspaces")
    if err != nil{
      log.Fatal(err)
    }

    defer rows.Close()

    for rows.Next() {
      err:=rows.Scan(&id,&name,&token)
      if err != nil {
        log.Fatal(err)
      }
      ctx.StatusCode(iris.StatusOK)
      ctx.JSON(iris.Map{"message": "You need a valid token!"})
      return
      // ctx.JSON({"status": 400, "message": token}, "400")

      // ctx.JSON()
      // ctx.StatusCode()

    }

    err = rows.Err()
    if err != nil {
      log.Fatal(err)
    }

    defer db.Close()

    // var slackworkspace string;
    // var id string;
    // err = db.QueryRow("SELECT id, name, token FROM workspaces").Scan(&id, &name, &token)

    // if err != nil {
    //   fmt.Println(err)
    // }

    // fmt.Println(slackworkspace);
    // response := SlackWorkspace{Id: id, Error: "false", Name: name, Token: token}
    // ctx.JSON(iris.Map{
    //   "status": 200,
    //   "message": response,
    // })

  })

  // Handling POST /prices
  app.Post("/prices", func(ctx iris.Context) {
		slack := Slack{}
    err := ctx.ReadForm(&slack)

		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
		}

    coin := strings.ToLower(slack.Text)

    // If a coin is specified, return data for that coin
    if (len(coin) > 0) && (coin == "btc" || coin == "eth" || coin == "xrp" || coin == "ltc") {

      if coin == "btc" {
        coinText = "Bitcoin"
        coinColor = "#F2A900"
      } else if coin == "eth" {
        coinText = "Ethereum"
        coinColor = "#3C3C3D"
      } else if coin == "xrp" {
        coinText = "Ripple"
        coinColor = "#0091CF"
      } else if coin == "ltc" {
        coinText = "Litecoin"
        coinColor = "#B6B6B6"
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

      for i := range cryptocurrencyData.Payload {
        if cryptocurrencyData.Payload[i].Book == "btc_mxn" {
          coinText = "Bitcoin"
          responseText = "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.Payload[i].Last + " MXN \n"
        } else if cryptocurrencyData.Payload[i].Book == "eth_mxn" {
          coinText = "Ethereum"
          responseText += "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.Payload[i].Last + " MXN \n"
        } else if cryptocurrencyData.Payload[i].Book == "xrp_mxn" {
          coinText = "Ripple"
          responseText += "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.Payload[i].Last + " MXN \n"
        } else if cryptocurrencyData.Payload[i].Book == "ltc_mxn" {
          coinText = "Litecoin"
          responseText += "*" + strings.ToUpper(coinText) + "*: $ " + cryptocurrencyData.Payload[i].Last + " MXN"
        }
      }

      if err != nil {
        log.Fatal(err)
      }

      ctx.JSON(iris.Map{
        "text": "The latest prices of BTC, ETH, XRP and LTC from Bitso are:",
        "response_type": "in_channel",
        "mrkdwn": true,
        "attachments":[]iris.Map{
          iris.Map{
            "title": "Start trading!",
            "title_link": "https://bitso.com/trade",
            "text": responseText,
            "color": coinColor,
          },
        },
      })
    }
	})

	app.Run(iris.Addr(":3333"))
}
