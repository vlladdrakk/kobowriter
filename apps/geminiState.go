package gbrowser

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"time"
)

type SaveState struct {
	History []string
}

func (s *GeminiBrowser) SaveState() {
	f, err := os.Create(s.SaveLocation + "/gemini.state")
	if err != nil {
		log.Println("Couldn't open file: ", err)
	}
	defer f.Close()

	json := s.toJson()
	_, err = f.Write(json)

	if err != nil {
		log.Println("Write failed: ", err)
	}

	log.Println("Write complete")
}

func LoadState(saveLocation string) (GeminiBrowser, bool) {
	fileLocation := saveLocation + "/gemini.state"
	if _, e := os.Stat(fileLocation); errors.Is(e, os.ErrNotExist) {
		log.Println("State file not found, continuing")
		return GeminiBrowser{}, false
	}

	rawBytes, err := os.ReadFile(fileLocation)
	if err != nil {
		log.Println("Couldn't read file: ", err)
		return GeminiBrowser{}, false
	}

	var data map[string]interface{}
	err = json.Unmarshal(rawBytes, &data)

	if err != nil {
		log.Println("Failed to unmarshal saved state, continuing. Error: ", err)
		return GeminiBrowser{}, false
	}

	browser := GeminiBrowser{}

	// Parse the currentPage
	browser.CurrentPage = pageFromJson(data["current_page"].(map[string]interface{}))

	if data["history"] != nil {
		for _, p := range data["history"].([]interface{}) {
			browser.History = append(browser.History, historyItemFromJson(p.(map[string]interface{})))
		}
	}

	if data["future"] != nil {
		for _, p := range data["future"].([]interface{}) {
			browser.Future = append(browser.Future, historyItemFromJson(p.(map[string]interface{})))
		}
	}

	browser.Cache = make(map[string]Page)
	for k, v := range data["cache"].(map[string]interface{}) {
		browser.Cache[k] = pageFromJson(v.(map[string]interface{}))
	}

	for _, b := range data["bookmarks"].([]interface{}) {
		browser.Bookmarks = append(browser.Bookmarks, bookmarkFromJson(b.(map[string]interface{})))
	}

	log.Println("browser", browser)
	return browser, true
}

func (s *GeminiBrowser) toJson() []byte {
	res, err := json.Marshal(s)

	if err != nil {
		log.Println("Error marshalling state to json", err)
		return nil
	}

	return res
}

func pageFromJson(json map[string]interface{}) Page {
	expTime, err := time.Parse(time.RFC3339Nano, json["expiration"].(string))
	if err != nil {
		log.Println("Failed to expiration parse time. Error: ", err)
		expTime = time.Now()
	}

	page := Page{
		Url: json["url"].(string),
		Exp: expTime,
		Position: Position{
			X: int(json["position"].(map[string]interface{})["X"].(float64)),
			Y: int(json["position"].(map[string]interface{})["Y"].(float64)),
		},
		View: hyperTextViewFromJson(json["view"].(map[string]interface{})),
	}

	return page
}

func hyperTextViewFromJson(json map[string]interface{}) HyperTextView {
	var wrapContent []string
	for _, s := range json["wrap_content"].([]interface{}) {
		wrapContent = append(wrapContent, s.(string))
	}

	linkMap := make(map[int]string)
	for k, v := range json["linkMap"].(map[string]interface{}) {
		key, err := strconv.Atoi(k)
		if err != nil {
			log.Println("Failed to parse line number for ", v.(string))
			continue
		}

		linkMap[key] = v.(string)
	}

	var lineCount []int
	for _, l := range json["lineCount"].([]interface{}) {
		lineCount = append(lineCount, int(l.(float64)))
	}

	view := HyperTextView{
		Width:       int(json["width"].(float64)),
		Height:      int(json["height"].(float64)),
		CursorIndex: int(json["cursor_index"].(float64)),
		CursorPos: Position{
			X: int(json["cursor_pos"].(map[string]interface{})["X"].(float64)),
			Y: int(json["cursor_pos"].(map[string]interface{})["Y"].(float64)),
		},
		LinkMap:     linkMap,
		Content:     json["content"].(string),
		WrapContent: wrapContent,
		LineCount:   lineCount,
	}

	return view
}

func historyItemFromJson(json map[string]interface{}) HistoryItem {
	return HistoryItem{
		Url: json["url"].(string),
		Position: Position{
			X: int(json["Position"].(map[string]interface{})["X"].(float64)),
			Y: int(json["Position"].(map[string]interface{})["Y"].(float64)),
		},
	}
}

func bookmarkFromJson(json map[string]interface{}) Bookmark {
	return Bookmark{
		Url:  json["url"].(string),
		Name: json["name"].(string),
	}
}
