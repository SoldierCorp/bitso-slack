package main

import (
	"encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"
)

type Response struct {
	Cryptocurrency `json:"payload"`
}

type Cryptocurrency struct {
	Last int `json:"last"`
}

// var (
//   token string
// )

// type LastValue struct {
// 	Last string `json:"last"`
// }

func getCryptocurrency(w http.ResponseWriter, r *http.Request) {

  if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// if token != r.FormValue("token") {
	// 	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	// 	return
  // }

  err := r.ParseForm()
  if err != nil {
    return
  }

  text := strings.ToLower(r.PostFormValue("text"))


  if (len(text) > 0) && (text == "btc" || text == "eth" || text == "xrp") {
    log.Print("Coin specified!");

    response, err := http.Get("https://api.bitso.com/v3/ticker/?book=" + text +"_mxn")

    if err != nil {
      fmt.Print(err.Error())
      os.Exit(1)
    }

    var responseObject Response
    responseData, err := ioutil.ReadAll(response.Body)

    if err != nil {
      log.Fatal(err)
    }

    json.Unmarshal(responseData, &responseObject)

    log.Print(responseObject);

    jsonResp, _ := json.Marshal(struct {
      Type string `json:"response_type"`
      Text string `json:"text"`
    }{
      Type: "in_channel",
      Text: text,
    })

    w.Header().Set("Content-Type", "application/json");
    fmt.Fprintf(w, string(jsonResp))
  } else {
    log.Print("Coin not specified or different term wrote!");

    response, err := http.Get("http://api.bitso.com/v3/ticker")

    log.Print(response.Body);
    if err != nil {
      fmt.Print(err.Error())
      os.Exit(1)
    }

    var body struct {
      Payload map[string]string `json:"payload"`
      Success  string `json:"success"`
    }

    json.NewDecoder(response.Body).Decode(&body)
    fmt.Println(body)
  }
}

func main() {

  http.HandleFunc("/prices", getCryptocurrency)

	err := http.ListenAndServe(":8080", nil) // setting listening port
  if err != nil {
      log.Fatal("ListenAndServe: ", err)
  }
}
