package internal

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

type headers map[string]string

var (
	// DateStartTatSaMon contains timestamps of tatsamon started date
	DateStartTatSaMon = time.Now().Unix()
	// Host points to the sailabove infrastructure wanted
	Host = ""
	// HostwihoutScheme points to the sailabove infrastructure wanted, without https://
	HostwihoutScheme = ""
	// User of sailabove to use
	User = ""
	// Password of sailabove account to use
	Password = ""
	// SailaboveAuth Basic Auth
	SailaboveAuth = ""
	// ConfigDir points to the Docker configuration directory
	ConfigDir string
	// Format to use for output. One of 'json', 'yaml', 'pretty'
	Format string
	// Home fetches the user home directory
	Home = os.Getenv("HOME")
)

// ReadConfig decode sailabove_auth and set user / password
func ReadConfig() error {

	if viper.GetString("sailabove_auth") == "" {
		return fmt.Errorf("Invalid Sailabove Authenticaton, see flag --sailabove-auth")
	}

	if viper.GetString("sailabove_host") == "" {
		return fmt.Errorf("Invalid Sailabove Host, see flag --sailabove-host")
	}

	Host = viper.GetString("sailabove_host")

	err := expandRegistryURL()
	if err != nil {
		return err
	}

	dataB, err := base64.StdEncoding.DecodeString(viper.GetString("sailabove_auth"))
	if err != nil {
		return fmt.Errorf("Error while decoding sailabove Auth: %s", err.Error())
	}

	data := string(dataB)
	if !strings.Contains(data, ":") {
		return fmt.Errorf("Invalid sailabove auth, does not contains ':'")
	}

	s := strings.Split(data, ":")
	User = s[0]
	Password = data[len(User)+1:]

	if User == "" || Password == "" {
		return fmt.Errorf("Missing user, password or host in configuration")
	}

	return nil
}

func expandRegistryURL() error {
	r, _ := regexp.Compile("http.?://")
	HostwihoutScheme = r.ReplaceAllString(Host, "")
	log.Debugf("Computed HostwihoutScheme: %s", HostwihoutScheme)

	if strings.Contains(Host, "/v1") == false {
		Host = Host + "/v1"
	}

	if strings.HasPrefix(Host, "http") || strings.HasPrefix(Host, "https") {
		return nil
	}
	pingOk, err := ping("https://" + Host)
	if err != nil {
		return err
	}
	if pingOk {
		Host = "https://" + Host
		return nil
	}

	Host = "http://" + Host
	return nil
}

func ping(hostname string) (bool, error) {
	urlPing := hostname + "/_ping"

	log.Debugf("Try ping on %s", urlPing)
	req, _ := http.NewRequest("GET", urlPing, nil)
	initRequest(req)
	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		log.Debugf("Ping OK on %s", urlPing)
		return true, nil
	}

	log.Debugf("Ping KO on %s", urlPing)
	return false, nil
}
