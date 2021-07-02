package scrapper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type Requester struct {
	lastTime time.Time
}

const Delay = time.Millisecond * 10

func NewRequester() *Requester {
	return &Requester{
		lastTime: time.Now(),
	}
}

func (r *Requester) GetData(url string) ([]byte, error) {
	if r.lastTime.After(time.Now()) {
		time.Sleep(r.lastTime.Add(time.Second).Sub(time.Now()))
	}

	resp, err := http.Get(url)
	r.lastTime = r.lastTime.Add(Delay)
	if err != nil {
		return nil, fmt.Errorf("Could not load data. Error: %v", err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Could not read response body. Error: %v", err)
	}

	return data, nil
}
