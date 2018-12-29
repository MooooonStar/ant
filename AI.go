package ant

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func Reply(msg string) (string, error) {
	url := fmt.Sprintf("http://api.qingyunke.com/api.php?key=free&appid=0&msg=%s", msg)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var Data struct {
		Result  int    `json:"result"`
		Content string `json:"content"`
	}
	err = json.Unmarshal(bt, &Data)
	if Data.Result != 0 {
		return "", fmt.Errorf(Data.Content)
	}
	return Data.Content, nil
}
