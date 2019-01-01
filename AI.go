package ant

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	Endpoint  = "http://openapi.tuling123.com/openapi/api/v2"
	ApiKey    = "b5d84a707cc3490a960b4e8e5d237a37"
	ApiSecret = "ca0f8d7a080371e3"
	UserID    = "373693"
	tlKey     = "a5052a22b8232be1e387ff153e823975"
)

func Encode(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

func Reply(msg string) (string, error) {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("http://www.tuling123.com/openapi/api?key=%s&info=%s", tlKey, url.QueryEscape(msg))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var Data struct {
		Code int    `json:"code"`
		Text string `json:"text"`
	}

	err = json.Unmarshal(bt, &Data)
	return Data.Text, err
}

// func Reply(msg string) (string, error) {
// 	client := http.Client{
// 		Timeout: 10 * time.Second,
// 	}

// 	payload := make(map[string]interface{}, 0)
// 	payload["reqType"] = 0
// 	text := make(map[string]interface{}, 0)
// 	text["inputText"] = map[string]interface{}{
// 		"text": msg,
// 	}
// 	tmp, _ := json.Marshal(text)

// 	h := md5.New()
// 	h.Write(tmp)
// 	encodedText := hex.EncodeToString(h.Sum(nil))

// 	payload["perception"] = encodedText
// 	payload["userInfo"] = map[string]interface{}{
// 		"apiKey": Encode(ApiKey),
// 		"userId": Encode(UserID),
// 	}

// 	body, err := json.Marshal(payload)
// 	if err != nil {
// 		return "", err
// 	}

// 	req, err := http.NewRequest("POST", Endpoint, bytes.NewReader(body))
// 	if err != nil {
// 		return "", err
// 	}
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("charset", "UTF-8")
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	bt, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", err
// 	}
// 	log.Println("resp", string(bt))

// 	var Resp struct {
// 		Data struct {
// 			Results []struct {
// 				ResultType string            `json:"resultType"`
// 				Values     map[string]string `json:"values"`
// 			} `json:"results"`
// 			Intent struct {
// 				Code int `json:"code"`
// 			} `json:"intent"`
// 		} `json:"data"`
// 	}
// 	err = json.Unmarshal(bt, &Resp)
// 	if err != nil {
// 		return "", err
// 	}
// 	log.Println("data", Resp)
// 	var answer string
// 	for _, result := range Resp.Data.Results {
// 		if result.ResultType == "text" {
// 			answer = result.Values["text"]
// 			break
// 		}
// 	}
// 	return answer, err
// }
