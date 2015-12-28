package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/christophwitzko/go-curl"
	"github.com/spf13/viper"

	"github.com/gin-gonic/gin"

	"github.com/ovh/al2tat/models"
	"github.com/ovh/tatsamon/internal"
	"github.com/ovh/tatsamon/utils"
	cron "gopkg.in/robfig/cron.v2"
)

// SailaboveController ...
type SailaboveController struct{}

type applicationsJSON struct {
	DateCurrent  int64
	Errors       []string                `json:"errors"`
	Warnings     []string                `json:"warnings"`
	Applications []string                `json:"applications"`
	Services     map[string]*ServiceJSON `json:"services"`
}

// ServiceJSON ...
type ServiceJSON struct {
	ApplicationName string          `json:"applicationName"`
	Name            string          `json:"serviceName"`
	PreviousState   string          `json:"previousState"`
	State           string          `json:"state"`
	Containers      []ContainerJSON `json:"containers"`
	WithPredictor   bool
}

// ServiceJSONCheck only for check container_network["predictor"] = true
type ServiceJSONCheck struct {
	ContainerNetwork struct {
		Predictor bool `json:"predictor"`
	} `json:"container_network"`
}

// ContainerJSON ...
type ContainerJSON struct {
	Command        []string    `json:"command"`
	CreationDate   string      `json:"creation_date"`
	DeploymentDate string      `json:"deployment_date"`
	Environment    []string    `json:"environment"`
	Hostname       string      `json:"hostname"`
	Image          string      `json:"image"`
	Name           string      `json:"name"`
	Network        interface{} `json:"network"`
	Repository     string      `json:"repository"`
	RepositoryTag  string      `json:"repository_tag"`
	RestartPolicy  interface{} `json:"restart_policy"`
	Service        string      `json:"service"`
	State          string      `json:"state"`
	User           string      `json:"user"`
	Workdir        string      `json:"workdir"`
	IPPrivate      string      `json:"ipPrivate"`
	IPPredictor    string      `json:"ipPredictor"`
}

var currentApps *applicationsJSON
var tatCron *cron.Cron
var sailaboveState string
var sailaboveLastSendAlert int64
var tatLastSendAlert int64
var cronCheckSailaboveSeconds int
var servicesPathToCheck map[string]string
var servicesPreviousState map[string]string
var servicesLastSendAlert map[string]int64

// ExcludeServices contains excluded services check
var ExcludeServices []string

// IncludeOnlyServices contains only services check.
var IncludeOnlyServices []string

// ServicesNoHTTP contains services with no check HTTP to do (only Sailabove check)
var ServicesNoHTTP []string

// InitCron init cron for tatsamon
// It's not a scalable service if cron is activated !
func InitCron() {
	cronCheckSailaboveSeconds = viper.GetInt("cron_check_sailabove")

	servicesPathToCheck = make(map[string]string)
	servicesPreviousState = make(map[string]string)
	servicesLastSendAlert = make(map[string]int64)

	err := internal.ReadConfig()
	if err != nil {
		log.Fatalf("Error with readConfig SA: %s", err.Error())
	}

	if viper.GetString("services_path_to_check") != "" {
		tuple := strings.Split(viper.GetString("services_path_to_check"), ",")
		for _, sp := range tuple {
			svcPath := strings.Split(sp, ":")
			if len(svcPath) >= 2 {
				// sp[len(svcPath[4]):] and not svcPath[1] to allow
				// define port with path : service::8080/path for port 8080 and path /path
				servicesPathToCheck[svcPath[0]] = sp[len(svcPath[0])+1:]
				log.Debugf("Init %s with a check on %s", svcPath[0], sp[len(svcPath[0])+1:])
			}
		}
	}

	if viper.GetBool("activate_cron") {
		log.Debugf("Init Cron")
		tatCron = cron.New()
		crontabContainers := fmt.Sprintf("*/%d * * * * *", viper.GetInt("cron_check_containers"))
		tatCron.AddFunc(crontabContainers, func() {
			if !utils.IsValidTatUser(viper.GetString("tat_username"), viper.GetString("tat_password")) {
				if time.Now().Unix()-tatLastSendAlert > int64(cronCheckSailaboveSeconds) {
					utils.SendAlertEmail("Tat Down !!!", "Check Tat Engine on tatmon tat_username and tat_password argument")
					tatLastSendAlert = time.Now().Unix()
				}
			} else {
				check(viper.GetString("tat_username"), viper.GetString("tat_password"))
			}
		})
		tatCron.Start()
	} else {
		log.Debugf("Cron desactivated with flag activate-cron")
	}
}

// ListApplications list applications, services and containers
func (*SailaboveController) ListApplications(ctx *gin.Context) {
	tatUsername := utils.GetHeader(ctx, utils.TatUsernameHeader)
	tatPassword := utils.GetHeader(ctx, utils.TatPasswordHeader)
	if utils.IsValidTatUser(tatUsername, tatPassword) {
		ctx.JSON(http.StatusOK, applicationsData(tatUsername, tatPassword))
	} else {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid tat user"})
	}
}

// CheckApplications ...
func (*SailaboveController) CheckApplications(ctx *gin.Context) {
	tatUsername := utils.GetHeader(ctx, utils.TatUsernameHeader)
	tatPassword := utils.GetHeader(ctx, utils.TatPasswordHeader)
	if !utils.IsValidTatUser(tatUsername, tatPassword) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "invalid tat user"})
		return
	}

	errors := check(tatUsername, tatPassword)
	if len(errors) > 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"errors": errors})
	} else {
		ctx.JSON(http.StatusOK, "")
	}
}

func check(tatUsername, tatPassword string) []string {
	log.Debugf(fmt.Sprintf("Check with %s", tatUsername))
	initCurrentApps(tatUsername, tatPassword)

	errors := []string{}

	for _, service := range currentApps.Services {

		if _, ok := servicesPreviousState[service.Name]; !ok {
			servicesPreviousState[service.Name] = ""
		}

		summary := ""
		service.State = "UP"
		nbAL := 0
		predictorRequested := false
		for _, container := range service.Containers {

			if len(ServicesNoHTTP) > 0 && utils.ArrayContains(ServicesNoHTTP, container.Service) {
				if container.State != "running" {
					errors = append(errors, fmt.Sprintf("Container %s is not running. Current state: %s", container.Name, container.State))
					service.State = "AL"
					nbAL++
					summary += fmt.Sprintf("#host:%s #state:%s", container.Hostname, container.State)
				} else {
					log.Debugf("Ok, container %s is running", container.Name)
				}
			} else {
				errors, summaryContainer := getCurl(errors, service, container, false)
				summaryPredictor := ""
				summaryPredictorBug := ""
				if !predictorRequested {
					if container.IPPredictor != "" && service.WithPredictor {
						errors, summaryPredictor = getCurl(errors, service, container, true)
					} else if container.IPPredictor == "" && service.WithPredictor {
						summaryPredictorBug = fmt.Sprintf("No IP Predictor on container %s, Please contact sailabove team.", container.Name)
					} else if container.IPPredictor == "" && !service.WithPredictor {
						summaryPredictorBug = fmt.Sprintf("IP Predictor on container %s on service with no predictor, Please contact sailabove team.", container.Name)
					}
					predictorRequested = true
				}
				if summaryPredictorBug != "" || summaryContainer != "" || summaryPredictor != "" {
					nbAL++
					service.State = "AL"
					summary += summaryPredictorBug + summaryPredictor + summaryContainer
				}
			}
		}

		now := time.Now().Unix()
		log.Debugf("service:%s previous:%s now:%s summary:%s", service.Name, servicesPreviousState[service.Name], service.State, summary)
		if summary != "" || servicesPreviousState[service.Name] != service.State {
			item := fmt.Sprintf("%s/%s/%s", internal.User, service.Name, internal.RemoveHTTPPrefix(internal.Host))
			al := &models.Alert{
				Alert:   service.State,
				NbAlert: int64(nbAL),
				Service: viper.GetString("alert_service"),
				Summary: fmt.Sprintf("#tatmon #sailabove:%s #account:%s #service:%s %s", internal.RemoveHTTPPrefix(internal.Host), internal.User, service.Name, summary),
			}
			// If service change status and service is UP before 45s after tatsamon started, don't send an al
			if service.State == "UP" && now-int64(internal.DateStartTatSaMon) < 45 { // 45s
				log.Debugf("TatSaMon starting and UP found -> no AL UP, but send Monitoring event")
				go utils.SendEventMonitoring(tatUsername, tatPassword, al, item)
				continue
			}
			delay := now - servicesLastSendAlert[service.Name]
			if servicesPreviousState[service.Name] == service.State && delay < int64(cronCheckSailaboveSeconds) { // 10min
				log.Debugf("State Already sent before 10min ago (%d seconds)", delay)
				continue
			}

			servicesLastSendAlert[service.Name] = time.Now().Unix()
			log.Debugf("service:%s, send new Alert summary:%s item:%s", service.Name, summary, item)
			go utils.SendEventAlOrMon(tatUsername, tatPassword, al, item)
		} else {
			log.Debugf("service:%s previous:%s now:%s, same state or no summary to send - continue", service.Name, servicesPreviousState[service.Name], service.State)
		}
		servicesPreviousState[service.Name] = service.State
	}
	return errors
}

func getCurl(errors []string, service *ServiceJSON, container ContainerJSON, withPredictorURL bool) ([]string, string) {
	predictor := ""
	summary := ""
	hostname := container.Hostname
	url, urlPredictor, hostnamePredictor := computeURL(service, container)
	if withPredictorURL {
		url = urlPredictor
		predictor = "#predictor"
		hostname = hostnamePredictor
	}

	log.Debugf("curl GET %s with url: %s", predictor, url)
	err, str, resp := curl.String("http://"+url, "dialtimeout=", viper.GetInt("dial_timeout"), "readtimeout=", viper.GetInt("read_timeout"), "deadline=", viper.GetInt("dead_line"))
	if err != nil {
		summary = fmt.Sprintf("#host:%s #state:%s err:%s %s url:%s ", hostname, container.State, err.Error(), predictor, url)
	} else if resp != nil && resp.StatusCode != http.StatusOK {
		summary = fmt.Sprintf("#host:%s #state:%s #http:%d %s url:%s ", hostname, container.State, resp.StatusCode, predictor, url)
	}

	if summary != "" {
		errors = append(errors, summary)
		return errors, summary
	}
	log.Debugf("Response OK from %s: %s - resp:", url, str, resp)
	return errors, ""
}

func computeURL(service *ServiceJSON, container ContainerJSON) (string, string, string) {
	urlPrivate := ""
	urlPredictor := ""
	hostnamePredictor := ""

	if _, ok := servicesPathToCheck[container.Service]; ok {
		urlPrivate = container.IPPrivate + servicesPathToCheck[container.Service]
		log.Debugf("Compute Url from --services-path-to-check arg : %s", urlPrivate)
	} else {
		urlPrivate = container.IPPrivate + viper.GetString("default_path_to_check")
	}

	if container.IPPredictor != "" {
		hostnamePredictor = fmt.Sprintf("%s.%s.app.%s", service.Name, service.ApplicationName, internal.HostwihoutScheme)
		urlPredictor = hostnamePredictor
		r, _ := regexp.Compile("^:[0-9]+")
		if _, ok := servicesPathToCheck[container.Service]; ok {
			// Predictor always on 80, remove port :9090/version gives /version
			urlPredictor += r.ReplaceAllString(servicesPathToCheck[container.Service], "")
			log.Debugf("Compute Url from --services-path-to-check arg : %s", urlPredictor)
		} else {
			urlPredictor += r.ReplaceAllString(viper.GetString("default_path_to_check"), "")
		}
		log.Debugf("Computed urlPredictor: %s", urlPredictor)
	}
	return urlPrivate, urlPredictor, hostnamePredictor
}

func computeIP(container ContainerJSON) (string, string) {
	ipPrivate := ""
	ipPredictor := ""
	for network, v := range container.Network.(map[string]interface{}) {
		if strings.Contains(network, "private") || strings.Contains(network, "predictor") {
			for keyIP, valIP := range v.(map[string]interface{}) {
				if keyIP == "ip" {
					if network == internal.User+"/private" {
						ipPrivate = valIP.(string)
						log.Debugf("Computed ip from Private %s for %s, hostname %s (%s)", ipPrivate, container.Service, container.Hostname, container.Name)
					} else if strings.Contains(network, "predictor") {
						ipPredictor = valIP.(string)
						log.Debugf("Computed ip from Predictor %s for %s, hostname %s (%s)", ipPredictor, container.Service, container.Hostname, container.Name)
					}
				}
			}
		}
	}
	return ipPrivate, ipPredictor
}

func initCurrentApps(tatUsername, tatPassword string) {
	if currentApps == nil || time.Now().Unix()-currentApps.DateCurrent > int64(cronCheckSailaboveSeconds) || sailaboveState == "AL" {
		log.Debugf("Refresh currentApps from sailabove")
		currentApps = applicationsData(tatUsername, tatPassword)
	}
}

func applicationsData(tatUsername, tatPassword string) *applicationsJSON {
	containers := []string{}

	apps, err := internal.GetListApplications(nil)
	if err != nil {
		if sailaboveState != "AL" || (sailaboveState == "AL" && time.Now().Unix()-sailaboveLastSendAlert > int64(cronCheckSailaboveSeconds)) {
			summary := fmt.Sprintf("#tatmon #sailabove:%s #account:%s #service:sailabove API Down", internal.RemoveHTTPPrefix(internal.Host), internal.User)
			al := &models.Alert{
				Alert:   "AL",
				NbAlert: 1,
				Service: viper.GetString("alert_service"),
				Summary: summary,
			}
			go utils.SendEventAlOrMon(tatUsername, tatPassword, al, "")
			log.Infof("Sailabove, send %s", summary)
		} else {
			log.Infof("Sailabove Alert already sent since 10min")
		}
		sailaboveLastSendAlert = time.Now().Unix()
		sailaboveState = "AL"
	} else {
		if sailaboveState == "AL" {
			summary := fmt.Sprintf("#tatmon #sailabove:%s #account:%s #service:sailabove OK", internal.RemoveHTTPPrefix(internal.Host), internal.User)
			al := &models.Alert{
				Alert:   "UP",
				NbAlert: 1,
				Service: viper.GetString("alert_service"),
				Summary: summary,
			}
			go utils.SendEventAlOrMon(tatUsername, tatPassword, al, "")
			log.Infof("Sailabove, send %s", summary)
			sailaboveLastSendAlert = time.Now().Unix()
		}
		sailaboveState = "UP"
	}

	a := &applicationsJSON{
		DateCurrent:  time.Now().Unix(),
		Applications: apps,
		Services:     make(map[string]*ServiceJSON),
	}

	for _, app := range apps {
		b, err := internal.ReqWant("GET", http.StatusOK, fmt.Sprintf("/applications/%s/containers", app), nil)
		if err != nil {
			a.Errors = append(a.Errors, err.Error())
			log.Errorf("Error Requesting sailabove /containers: %s", err.Error())
			continue
		}
		err = json.Unmarshal(b, &containers)
		if err != nil {
			a.Errors = append(a.Errors, err.Error())
			log.Errorf("Error unmarshal apps: %s", err.Error())
			continue
		}

		for _, containerID := range containers {
			var container ContainerJSON
			b, err := internal.ReqWant("GET", http.StatusOK, fmt.Sprintf("/applications/%s/containers/%s", app, containerID), nil)
			if err != nil {
				a.Errors = append(a.Errors, err.Error())
				log.Errorf("Error Requesting sailabove /containers: %s", err.Error())
				continue
			}
			err = json.Unmarshal(b, &container)
			if err != nil {
				a.Errors = append(a.Errors, err.Error())
				log.Errorf("Error unmarshal container: %s", err.Error())
				continue
			}

			if utils.ArrayContains(ExcludeServices, container.Service) {
				log.Debugf("%s excluded from configuration (with exclude-services flag)", container.Service)
				continue
			}

			if len(IncludeOnlyServices) > 0 && !utils.ArrayContains(IncludeOnlyServices, container.Service) {
				log.Debugf("%s excluded from configuration (with include-only-services flag)", container.Service)
				continue
			}

			if container.State != "running" {
				a.Warnings = append(a.Warnings, fmt.Sprintf("%s of %s is not running", container.Name, container.Service))
			}

			container.IPPrivate, container.IPPredictor = computeIP(container)
			if container.IPPrivate == "" {
				a.Warnings = append(a.Warnings, fmt.Sprintf("%s of %s is not running", container.Name, container.Service))
				log.Warnf("%s have no IP Private in same network as tatsamon - excluded", container.Service)
				continue
			}

			envs := []string{}
			for _, env := range container.Environment {
				if strings.Contains(env, "PASS") || strings.Contains(env, "TOKEN") || strings.Contains(env, "KEY") || strings.Contains(env, "AUTH") {
					s := strings.Split(env, "=")
					if len(s) > 0 {
						envs = append(envs, s[0]+"=****")
					}
					continue
				}
				envs = append(envs, env)
			}
			container.Environment = envs

			if _, ok := a.Services[container.Service]; !ok {

				bs, err := internal.ReqWant("GET", http.StatusOK, fmt.Sprintf("/applications/%s/services/%s", app, container.Service), nil)
				fmt.Printf("bs:%s", string(bs))
				if err != nil {
					a.Errors = append(a.Errors, err.Error())
					log.Errorf("Error Requesting sailabove /services: %s", err.Error())
					continue
				}

				var serviceCheck ServiceJSONCheck
				err = json.Unmarshal(bs, &serviceCheck)
				if err != nil {
					a.Errors = append(a.Errors, err.Error())
					log.Errorf("Error unmarshal apps: %s", err.Error())
					continue
				}

				c := []ContainerJSON{container}
				a.Services[container.Service] = &ServiceJSON{
					ApplicationName: app,
					Name:            container.Service,
					Containers:      c,
					WithPredictor:   serviceCheck.ContainerNetwork.Predictor,
				}
			} else {
				c := append(a.Services[container.Service].Containers, container)
				s := a.Services[container.Service]
				s.Containers = c
				a.Services[container.Service] = s
			}
		}
	}
	return a
}
