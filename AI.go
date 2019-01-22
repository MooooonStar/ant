package ant

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	tlKey = "a5052a22b8232be1e387ff153e823975"
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

	if err = json.Unmarshal(bt, &Data); err != nil {
		return "", err
	}
	if Data.Code != 100000 {
		return "", errors.New("Something is wrong")
	}
	return Data.Text, nil
}
