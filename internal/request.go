package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func initRequest(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "tatmon/"+VERSION)
	req.Header.Set("Connection", "close")
}

func getHTTPClient() *http.Client {
	tr := &http.Transport{}
	return &http.Client{Transport: tr}
}

// GetWantJSON GET on path and return []byte of JSON
func GetWantJSON(path string) ([]byte, error) {
	return ReqWant("GET", http.StatusOK, path, nil)
}

// ReqWantJSON requests with a method on a path, check wantCode and returns []byte of JSON
func ReqWantJSON(method string, wantCode int, path string, body []byte) ([]byte, error) {
	return ReqWant(method, wantCode, path, body)
}

// ReqWant requests with a method on a path, check wantCode and returns []byte
func ReqWant(method string, wantCode int, path string, jsonStr []byte) ([]byte, error) {
	return apiRequest(method, wantCode, path, jsonStr)
}

// apiRequest helper, issue the request, return full body or error
func apiRequest(method string, wantCode int, path string, jsonStr []byte) ([]byte, error) {
	bodyStream, code, err := doRequest(method, path, jsonStr)
	if err != nil {
		return nil, err
	}

	defer bodyStream.Close()

	var body []byte
	body, err = ioutil.ReadAll(bodyStream)
	if err != nil {
		return nil, err
	}

	// Hard-wire 201-200 equivalence to work around api returning 200 in place of 201
	if code != wantCode && !(wantCode == http.StatusCreated && code == http.StatusOK) {
		if err == nil {
			return nil, FormatOutputErrror(body)
		}
		return nil, err
	}
	return body, nil
}

// Request executes an authentificated HTTP request on $path given $method and $args
func Request(method string, path string, args []byte) ([]byte, int, error) {

	respBody, code, err := doRequest(method, path, args)
	if err != nil {
		return nil, 0, err
	}
	defer respBody.Close()

	var body []byte
	body, err = ioutil.ReadAll(respBody)
	if err != nil {
		return nil, 0, err
	}

	return body, code, nil
}

//doRequest builds the request and return io.ReadCloser
func doRequest(method string, path string, args []byte) (io.ReadCloser, int, error) {
	var req *http.Request
	if args != nil {
		req, _ = http.NewRequest(method, Host+path, bytes.NewReader(args))
	} else {
		req, _ = http.NewRequest(method, Host+path, nil)
	}
	initRequest(req)

	req.SetBasicAuth(User, Password)
	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}

	log.Debugf("Response Status: %s", resp.Status)
	log.Debugf("Request path: %s", Host+path)
	log.Debugf("Request Headers: %s", req.Header)
	log.Debugf("Request Body: %s", string(args))
	log.Debugf("Response Headers: %s", resp.Header)

	return resp.Body, resp.StatusCode, nil
}

// GetListApplications returns list of applications, GET on /applications
func GetListApplications(apps []string) ([]string, error) {
	if len(apps) == 0 {
		b, err := ReqWant("GET", http.StatusOK, "/applications", nil)
		if err != nil {
			return []string{}, err
		}
		err = json.Unmarshal(b, &apps)
		if err != nil {
			return []string{}, err
		}
	}
	return apps, nil
}
