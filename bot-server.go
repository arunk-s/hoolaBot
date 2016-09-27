package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
)

var answeredUpdates int64
var answerIDs = 1

type user struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastNname string `json:"last_name"`
	UserName  string `json:"username"`
}

type botResponse struct {
	Ok     bool                   `json:"ok"`
	Result map[string]interface{} `json:"result"`
	Desc   string                 `json:"description"`
	Error  int                    `json:"error_code"`
}

type botResponseUpd struct {
	Ok     bool                     `json:"ok"`
	Result []map[string]interface{} `json:"result"`
	Desc   string                   `json:"description"`
	Error  int                      `json:"error_code"`
}

func (u *user) parseUser(m map[string]interface{}) {
	if v, ok := m["id"].(float64); ok {
		u.ID = int64(v)
	}
	if v, ok := m["first_name"].(string); ok {
		u.FirstName = v
	}
	if v, ok := m["username"].(string); ok {
		u.UserName = v
	}
	if v, ok := m["last_name"].(string); ok {
		u.LastNname = v
	}
}

type botUpdate struct {
	ID                 int64                  `json:"update_integer"`
	Message            map[string]interface{} `json:"message"`
	EditedMessage      map[string]interface{} `json:"edited_message"`
	InlineQuery        map[string]interface{} `json:"inline_query"`
	ChosenInlineResult map[string]interface{} `json:"chosen_inline_result"`
	CallBackQuery      map[string]interface{} `json:"CallBackQuery"`
}

type inlineQuery struct {
	ID       int64                  `json:"id"`
	From     map[string]interface{} `json:"from"`
	Location map[string]interface{} `json:"location"`
	Query    string                 `json:"query"`
	Offset   string                 `json:"offset"`
}

func (i *inlineQuery) parseInline(m map[string]interface{}) {
	if v, ok := m["id"].(float64); ok {
		i.ID = int64(v)
	}
	if v, ok := m["from"].(map[string]interface{}); ok {
		i.From = v
	}
	if v, ok := m["location"].(map[string]interface{}); ok {
		i.Location = v
	}
	if v, ok := m["query"].(string); ok {
		i.Query = v
	}
	if v, ok := m["offset"].(string); ok {
		i.Offset = v
	}
}

type answerInline struct {
	ID     int64         `json:"inline_query_id"`
	Result []inlinePhoto `json:"results"`
}

type inlinePhoto struct {
	Type           string              `json:"type"`
	ID             int64               `json:"id"`
	PhotoURL       string              `json:"photo_url"`
	ThumbURL       string              `json:"thumb_url"`
	MessageContent inputMessageContent `json:"input_message_content"`
}

type inputMessageContent struct {
	Message   string `json:"message_text"`
	ParseMode string `json:"parse_mode"`
}

func main() {
	http.HandleFunc("/"+os.Getenv("BOT_TOKEN"), doAction)
	http.HandleFunc("/", handler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

func doAction(w http.ResponseWriter, r *http.Request) {
	var (
		boturl = "https://api.telegram.org/bot" + os.Getenv("BOT_TOKEN")
		upd    botUpdate
		from   user
		inline inlineQuery
		// inlineAns answerInline
		inlineResult []inlinePhoto
	)
	// defer r.Body.Close()
	rDecoder := json.NewDecoder(r.Body)
	err := rDecoder.Decode(&upd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// we have already returned the resposne
	if upd.ID < answeredUpdates {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	answeredUpdates = upd.ID
	if upd.InlineQuery != nil {

		inline.parseInline(upd.InlineQuery)
		if inline.From != nil {
			from.parseUser(inline.From)
			answer := makeMessage(from, inline.Query)
			inlineResult = append(inlineResult, answer)
			inlineByte, err := json.Marshal(inlineResult)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			values := url.Values{}
			values.Add("inline_query_id", fmt.Sprintf("\"%d\"", inline.ID))
			values.Add("results", string(inlineByte))
			resp, err := http.PostForm(boturl+"/answerInlineQuery", values)
			if err != nil {
				log.Printf("POST to telegram failed %v\n", err.Error())
				return
			}

			// fmt.Fprintf(w, "Hello, %v", values)
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				log.Printf("Success answeredQuery %v\n", upd.ID)
			} else {
				log.Printf("Failed answeredQuery %v\n", resp.Status)
			}
			return
		}
	}
	http.Error(w, "Only Inline query are Accepted", http.StatusNotImplemented)
	return
}

func makeMessage(from user, msg string) inlinePhoto {

	var (
		fixed  inlinePhoto
		answer string
	)
	if from.UserName != "" {
		answer = from.UserName + " " + msg
	} else {
		answer = from.FirstName + " " + msg
	}

	fixed.Type = "photo"
	fixed.ID = int64(answerIDs)
	fixed.MessageContent = inputMessageContent{
		Message: answer,
	}
	answerIDs++
	return fixed
}
