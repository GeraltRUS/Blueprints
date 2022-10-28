package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Структура отправляемого во внешний API запроса.
type BodyStruct struct {
	Event      string                 `json:"Event"`
	Message    string                 `json:"message"`
	Attributes map[string]interface{} `json:"Attributes"`
	Groups     []string               `json:"Groups"`
}

// Создание локального "сервера" для отправки на него POST ниже
func startServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "getHandler: incoming request %#v\n", r)
		fmt.Fprintf(w, "getHandler: r.Url %#v\n", r.URL)
	})
	http.HandleFunc("/raw_body", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprintf(w, "postHandler: raw body %s\n", string(body))
	})
	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}

// Функция создания и отправки запроса.
func runTransportAndPost() {
	var BodyS BodyStruct
	var c data

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}

	// Присваиваем полю message данные из файла
	c.getData()
	BodyS.Event = c.Event
	BodyS.Message = c.Text
	BodyS.Attributes = StrToMap(c.Attributes)

	//Читаем конфиг и обогощаем запрос
	url, apikey, groups := YamlConfRead()
	BodyS.Groups = groups

	// Блок для оборачивания в JSON
	alldata, err := json.MarshalIndent(&BodyS, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}
	body := bytes.NewBuffer(alldata)
	req, _ := http.NewRequest(http.MethodPost, url, body)
	req.Header.Add("accept", "*/*")
	req.Header.Add("X-API-KEY", apikey)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(BodyS.Message)))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Произошла ошибка:", err)
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	fmt.Printf("%#s\n\n\n", string(respBody))

}

// Структура и функция чтения из файла-конфига
type YmlConf struct {
	Endpoint string   `yaml:"Endpoint"`
	APIKey   string   `yaml:"ApiKey"`
	Groups   []string `yaml:"Groups"`
}

func YamlConfRead() (string, string, []string) {
	ymlFile, err := os.Open("conf.yml")
	if err != nil {
		panic(err)
	}
	defer ymlFile.Close()
	bytes, err := io.ReadAll(ymlFile)
	if err != nil {
		fmt.Println(err)
	}

	t := YmlConf{}
	error1 := yaml.Unmarshal(bytes, &t)
	if error1 != nil {
		fmt.Println(error1)
	}
	url := t.Endpoint
	apikey := t.APIKey
	groups := t.Groups
	return url, apikey, groups
}

// Структура и функция чтения из файла с данными
type data struct {
	Event      string `yaml:"Event"`
	Attributes string `yaml:"Attributes"`
	Text       string `yaml:"Text"`
}

func (c *data) getData() *data {
	ymlFile, err := os.Open("data.yml")
	if err != nil {
		panic(err)
	}

	yamlFile, err := io.ReadAll(ymlFile)
	if err != nil {
		fmt.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Printf("Unmarshal: %v", err)
	}

	return c
}

// Функция считывания атрибутов в карту
func StrToMap(in string) map[string]interface{} {
	res := make(map[string]interface{})
	array := strings.Split(in, "\n")
	temp := make([]string, 2)
	for _, val := range array {
		if val != "" {
			temp = strings.Split(val, ": ")
			res[temp[0]] = temp[1]
		}
	}
	return res
}

func main() {
	go startServer()
	time.Sleep(100 * time.Millisecond)
	runTransportAndPost()
	fmt.Println("Ok")
}
