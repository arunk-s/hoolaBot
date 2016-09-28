package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	ID                 int64                  `json:"update_id"`
	Message            map[string]interface{} `json:"message"`
	EditedMessage      map[string]interface{} `json:"edited_message"`
	InlineQuery        map[string]interface{} `json:"inline_query"`
	ChosenInlineResult map[string]interface{} `json:"chosen_inline_result"`
	CallBackQuery      map[string]interface{} `json:"callback_query"`
}

type inlineQuery struct {
	ID       string                 `json:"id"`
	From     map[string]interface{} `json:"from"`
	Location map[string]interface{} `json:"location"`
	Query    string                 `json:"query"`
	Offset   string                 `json:"offset"`
}

func (i *inlineQuery) parseInline(m map[string]interface{}) {
	if v, ok := m["id"].(string); ok {
		i.ID = (v)
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
	ID     string        `json:"inline_query_id"`
	Result []inlinePhoto `json:"results"`
}

type inlinePhoto struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	// PhotoID string `json:"photo_file_id"`
	PhotoURL       string              `json:"photo_url"`
	ThumbURL       string              `json:"thumb_url"`
	MessageContent inputMessageContent `json:"input_message_content"`
}

type inputMessageContent struct {
	Message               string `json:"message_text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type inputTextMessageContent struct {
	Text                  string `json:"message_text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

// InlineQueryResultArticle is an inline query response article.
type inlineQueryResultArticle struct {
	Type                string      `json:"type"`                            // required
	ID                  string      `json:"id"`                              // required
	Title               string      `json:"title"`                           // required
	InputMessageContent interface{} `json:"input_message_content,omitempty"` // required
	ReplyMarkup         interface{} `json:"reply_markup,omitempty"`
	URL                 string      `json:"url"`
	HideURL             bool        `json:"hide_url"`
	Description         string      `json:"description"`
	ThumbURL            string      `json:"thumb_url"`
	ThumbWidth          int         `json:"thumb_width"`
	ThumbHeight         int         `json:"thumb_height"`
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
		upd botUpdate
	)
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
		v, err := answerInlineQuery(upd.InlineQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// fmt.Fprintf(w, "Hello, %v", v)
		err = makeFormRequest("answerInlineQuery", v)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if upd.Message != nil {
		values := url.Values{}
		if v, ok := upd.Message["chat"].(map[string]interface{}); ok {
			chatID := int(v["id"].(float64))
			values.Add("chat_id", fmt.Sprintf("%d", chatID))
			values.Add("text", "I just do this.")
		}
		makeFormRequest("sendMessage", values)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("Unfullfilled update %v\n", upd)
	http.Error(w, "Only Inline query are Accepted", http.StatusNotImplemented)
	return
}

func makeMessage(ID string, from user, msg string) interface{} {

	var (
		// fixed inlinePhoto
		fixed  inlineQueryResultArticle
		answer string
	)
	if from.UserName != "" {
		answer = from.UserName + " " + msg
	} else {
		answer = from.FirstName + " " + msg
	}

	// fixed.Type = "photo"
	fixed.ID = ID
	// fixed.MessageContent = inputMessageContent{
	// 	Message: answer,
	// }
	fixed.InputMessageContent = inputTextMessageContent{
		Text: answer,
	}
	fixed.Type = "article"
	fixed.Title = answer
	// fixed.ID
	answerIDs++
	return fixed
}

func answerInlineQuery(m map[string]interface{}) (url.Values, error) {
	var (
		inline       inlineQuery
		from         user
		inlineResult []interface{}
		values       = url.Values{}
	)
	inline.parseInline(m)
	if inline.From != nil {
		from.parseUser(inline.From)
		answer := makeMessage(inline.ID, from, inline.Query)
		inlineResult = append(inlineResult, answer)
		inlineByte, err := json.Marshal(inlineResult)
		if err != nil {
			log.Printf("json marshal failed %v\n", err.Error())
			return values, err
		}
		values.Add("inline_query_id", inline.ID)
		values.Add("is_personal", strconv.FormatBool(true))
		values.Add("results", string(inlineByte))
		return values, nil
	}
	return values, nil
}

func makeFormRequest(method string, params url.Values) error {
	var (
		boturl = "https://api.telegram.org/bot" + os.Getenv("BOT_TOKEN")
	)
	resp, err := http.PostForm(boturl+"/"+method, params)
	if err != nil {
		log.Printf("Post to telegram failed %v\n", err.Error())
		return err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading response failed %v\n", err.Error())
		return err
	}

	if resp.StatusCode == 200 {
		log.Printf("Success: %v\n", method)
	} else {
		log.Printf("Unexpected %v \n", resp.Status)
		log.Printf("Data posted %v\n", params)
		log.Printf("Response %v", string(respBytes))
	}
	return nil
}
