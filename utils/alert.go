package utils

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/al2tat/models"
	"github.com/spf13/viper"
)

// ServicesEmail contains services on which we want to do dedicated email alerting instead of al2tat
var ServicesEmail []string

// SendEventAlOrMon send an Alert and /or an Event to Al2tat (alert or monitoring)
func SendEventAlOrMon(tatUsername, tatPassword string, alert *models.Alert, item string) {
	if viper.GetString("url_al2tat") == "" {
		log.Errorf("Invalid url_al2tat configuration")
		return
	}

	if viper.GetBool("activate_alerting") && len(ServicesEmail) > 0 && ArrayContains(ServicesEmail, item) {
		log.Debugf("Using dedicated email alerting for %s", item)
		subject := fmt.Sprintf("%s%d %s %s", alert.Alert, alert.NbAlert, alert.Service, item)
		log.Debugf("Sending dedicated email alerting with subject: %s", subject)
		err := sendEmailGeneric(templAlert, subject, viper.GetString("mail_alert_destination"), "")
		if err != nil {
			log.Errorf("Error while posting to dedicated email alerting (alert): %s", err.Error())

			s := "Please Check dedicated email alerting service. Tatmon try to send this:\n"
			s += fmt.Sprintf("state:%s%d\nservice:%s\nsummary:%s\n", alert.Alert, alert.NbAlert, alert.Service, alert.Summary)
			SendAlertEmail("Service dedicated email alerting seems down !!!", s)
		}
	} else if viper.GetBool("activate_alerting") {
		log.Debugf("Sending alert on Al2tat for %s", item)
		body, err := sendAlert(tatUsername, tatPassword, alert)
		if err != nil {
			log.Errorf("Error while posting to al2tat (alert): %s", err.Error())

			s := "Please Check Al2Tat service. Tatmon try to send this:\n"
			s += fmt.Sprintf("state:%s%d\nservice:%s\nsummary:%s\n", alert.Alert, alert.NbAlert, alert.Service, alert.Summary)
			s += fmt.Sprintf("\nReponse from al2tat:%s", string(body))
			SendAlertEmail("Service Al2Tat seems down !!!", s)
		}
	} else {
		log.Debugf("Send Event to AL2TAT alert is disabled by configuration")
	}

	if viper.GetBool("activate_monitoring") {
		if item != "" {
			SendEventMonitoring(tatUsername, tatPassword, alert, item)
		}
	} else {
		log.Debugf("Send Event to AL2TAT monitoring is disabled by configuration")
	}
}

// SendEventMonitoring send 'only' a monitoring event
func SendEventMonitoring(tatUsername, tatPassword string, alert *models.Alert, item string) {
	monitoring := &models.Monitoring{
		Status:  alert.Alert,
		Service: alert.Service,
		Item:    item,
		Summary: alert.Summary,
	}

	body, err := sendMonitoring(tatUsername, tatPassword, monitoring, item)

	if err != nil {
		log.Errorf("Error while posting to al2tat (Monitoring): %s", err.Error())
		s := "Please Check Al2Tat service. Tatmon try to send this:\n"
		s += fmt.Sprintf("state:%s\nservice:%s\nitem:%s\nsummary:%s\n", alert.Alert, alert.Service, item, alert.Summary)
		s += fmt.Sprintf("\nReponse from al2tat:%s", string(body))
		SendAlertEmail("Service Al2Tat seems down !!!", s)
	}
}

func sendMonitoring(tatUsername, tatPassword string, monitoring *models.Monitoring, item string) ([]byte, error) {
	if viper.GetString("tat_monitoring_topic") == "" {
		return nil, fmt.Errorf("Error, flag tat-monitoring-topic not setted")
	}

	j, err := json.Marshal(monitoring)
	if err != nil {
		return nil, fmt.Errorf("Error while Marshalling monitoring")
	}
	body, errPost := PostWant(
		viper.GetString("url_al2tat"),
		"/monitoring/sync",
		j,
		tatUsername,
		tatPassword,
		viper.GetString("tat_monitoring_topic"),
	)

	if errPost != nil {
		return body, errPost
	}
	return nil, nil
}

func sendAlert(tatUsername, tatPassword string, alert *models.Alert) ([]byte, error) {

	j, err := json.Marshal(alert)
	if err != nil {
		return nil, fmt.Errorf("Error while Marshalling alert")
	}
	body, errPost := PostWant(
		viper.GetString("url_al2tat"),
		"/alert/sync",
		j,
		tatUsername,
		tatPassword,
		viper.GetString("tat_alerts_topic"),
	)

	if errPost != nil {
		return body, errPost
	}
	return nil, nil
}
