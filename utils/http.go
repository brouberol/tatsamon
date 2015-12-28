package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

var sslInsecureSkipVerify bool
var url, username, password string

// Tat_username header
var TatUsernameHeader = "Tat_username"

// Tat_password header
var TatPasswordHeader = "Tat_password"

// Tat_topic header
var TatTopicHeader = "Tat_topic"

// GetHeader returns header value from request
func GetHeader(ctx *gin.Context, headerName string) string {
	h := strings.ToLower(headerName)
	hd := strings.ToLower(strings.Replace(headerName, "_", "-", -1))
	for k, v := range ctx.Request.Header {
		if strings.ToLower(k) == h {
			return v[0]
		} else if strings.ToLower(k) == hd {
			return v[0]
		}
	}
	return ""
}

func initRequest(req *http.Request, tatUsername, tatPassword, tatTopic string) {
	req.Header.Set(TatUsernameHeader, tatUsername)
	req.Header.Set(TatPasswordHeader, tatPassword)
	if tatTopic != "" {
		req.Header.Set(TatTopicHeader, tatTopic)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "close")
}

// GetWantBody checks is http status is equals to 200, and returns a GET request
func GetWantBody(url, path, tatUsername, tatPassword, tatTopic string) ([]byte, error) {
	return reqWant(url, "GET", http.StatusOK, path, nil, tatUsername, tatPassword, tatTopic)
}

// PostWant post a message to tat engine, checks if http code is equals to 201 and returns body
func PostWant(url, path string, jsonStr []byte, tatUsername, tatPassword, tatTopic string) ([]byte, error) {
	return reqWant(url, "POST", http.StatusCreated, path, jsonStr, tatUsername, tatPassword, tatTopic)
}

func isHTTPS(url string) bool {
	return strings.HasPrefix(url, "https")
}

func getHTTPClient(url string) *http.Client {
	var tr *http.Transport
	if isHTTPS(url) {
		tlsConfig := getTLSConfig()
		tr = &http.Transport{TLSClientConfig: tlsConfig}
	} else {
		tr = &http.Transport{}
	}

	return &http.Client{Transport: tr}
}

func getTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: sslInsecureSkipVerify,
	}
}

func reqWant(url, method string, wantCode int, path string, jsonStr []byte, tatUsername, tatPassword, tatTopic string) ([]byte, error) {
	if url == "" {
		return []byte{}, fmt.Errorf("Invalid URL")
	}

	requestPath := url + path
	var req *http.Request
	if jsonStr != nil {
		req, _ = http.NewRequest(method, requestPath, bytes.NewReader(jsonStr))
	} else {
		req, _ = http.NewRequest(method, requestPath, nil)
	}

	initRequest(req, tatUsername, tatPassword, tatTopic)
	resp, err := getHTTPClient(url).Do(req)
	if err != nil {
		log.Errorf("Error with getHTTPClient %s", err.Error())
		return []byte{}, fmt.Errorf("Error with getHTTPClient %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantCode {
		ret := fmt.Sprintf("Response Status:%s\n", resp.Status)
		ret += fmt.Sprintf("Request path :%s\n", requestPath)
		ret += fmt.Sprintf("Request :%s\n", string(jsonStr))
		ret += fmt.Sprintf("Response Headers:%s\n", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		ret += fmt.Sprintf("Response Body:%s\n", string(body))
		log.Errorf(ret)
		return []byte(ret), fmt.Errorf("Response code %d with Body:%s", resp.StatusCode, string(body))
	}
	log.Debugf("%s %s", method, requestPath)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error with ioutil.ReadAll %s", err.Error())
		return []byte{}, fmt.Errorf("Error with ioutil.ReadAll %s", err.Error())
	}
	return body, nil
}
