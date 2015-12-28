package main

import (
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/ovh/tatsamon/controllers"
	"github.com/ovh/tatsamon/routes"
	"github.com/ovh/tatsamon/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var mainCmd = &cobra.Command{
	Use:   "tatsamon",
	Short: "Tat Sailabove Monitoring",
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("tatsamon")
		viper.AutomaticEnv()

		if viper.GetBool("production") {
			// Only log the warning severity or above.
			log.SetLevel(log.WarnLevel)
			log.Info("Set Production Mode ON")
			gin.SetMode(gin.ReleaseMode)
		} else {
			log.SetLevel(log.DebugLevel)
		}

		if viper.GetString("log_level") != "" {
			switch viper.GetString("log_level") {
			case "debug":
				log.SetLevel(log.DebugLevel)
			case "info":
				log.SetLevel(log.InfoLevel)
			case "error":
				log.SetLevel(log.ErrorLevel)
			}
		}

		router := gin.Default()
		routes.InitRoutesCheck(router)
		if viper.GetBool("enabled_api_applications") {
			routes.InitRoutesApplications(router)
		}
		routes.InitRoutesSystem(router)

		controllers.InitCron()
		if viper.GetString("sailabove_auth") == "" {
			log.Fatalf("Invalid sailabove Auth. Please use sailabove_auth flag")
		} else if viper.GetString("alert_service") == "" {
			log.Fatalf("Invalid Service for Alert. Please use alert_service flag")
		} else if viper.GetString("tat_alerts_topic") == "" {
			log.Fatalf("Invalid Tat Topic to send alert. Please use tat_alerts_topic flag")
		} else if viper.GetBool("activate_cron") &&
			(viper.GetString("tat_username") == "" || viper.GetString("tat_password") == "") {
			log.Fatalf("Invalid Tat Credentials to communicate with tat. Please check tat-username and tat-password flags")
		} else {
			if viper.GetString("authorized_users") != "" {
				utils.AuthorizedUsers = strings.Split(viper.GetString("authorized_users"), ",")
				log.Debugf("Services concerned by authorized_users clause : %+v", utils.AuthorizedUsers)
			}
			if viper.GetString("exclude_services") != "" {
				controllers.ExcludeServices = strings.Split(viper.GetString("exclude_services"), ",")
				log.Debugf("Services concerned by exclude_services clause : %+v", controllers.ExcludeServices)
			}
			if viper.GetString("include_only_services") != "" {
				controllers.IncludeOnlyServices = strings.Split(viper.GetString("include_only_services"), ",")
				log.Debugf("Services concerned by include_only_services clause : %+v", controllers.IncludeOnlyServices)
			}
			if viper.GetString("services_no_http") != "" {
				controllers.ServicesNoHTTP = strings.Split(viper.GetString("services_no_http"), ",")
				log.Debugf("Services concerned by services_no_http clause : %+v", controllers.ServicesNoHTTP)
			}
			if viper.GetString("services_mail") != "" {
				utils.ServicesEmail = strings.Split(viper.GetString("services_mail"), ",")
				log.Debugf("Services concerned by services_mail clause : %+v", utils.ServicesEmail)
			}

			router.Run(":" + viper.GetString("listen_port"))
		}
	},
}

func init() {
	flags := mainCmd.Flags()
	flags.Bool("production", false, "Production mode")
	flags.String("log-level", "", "Log Level : debug, info or warn")
	flags.String("listen-port", "8086", "Tatsamon Listen Port")
	flags.String("url-tat-engine", "http://localhost:8080", "URL Tat Engine")
	flags.String("url-al2tat", "http://localhost:8082", "URL AL2Tat")
	flags.String("tat-username", "tat.system.tatsamon", "Tat Username")
	flags.String("tat-password", "", "Tat Password")
	flags.String("tat-alerts-topic", "/Internal/Alerts", "Tat Alerts Topic")
	flags.String("tat-monitoring-topic", "", "Tat Monitoring Topic")
	flags.String("sailabove-auth", "", "Sailabove Auth (base64)")
	flags.String("sailabove-host", "sailabove.io", "Sailabove Host")
	flags.String("alert-service", "", "Your Service for al2tat message")
	flags.Int("cron-check-sailabove", 600, "If activate-cron=true, seconds before each call to API Sailabove")
	flags.Int("cron-check-containers", 30, "If activate-cron=true, seconds before each call each containers")
	flags.Bool("enabled-api-applications", false, "Enable or not endpoint /applications")
	flags.Bool("activate-cron", true, "Activate internal tatsamon cron")
	flags.Bool("activate-alerting", true, "Activate Alerts Generation to Al2tat service")
	flags.Bool("activate-monitoring", true, "Activate Event Monitoring to Al2tat service")
	flags.Bool("no-smtp", false, "No SMTP mode")
	flags.String("smtp-host", "", "SMTP Host")
	flags.String("smtp-port", "", "SMTP Port")
	flags.Bool("smtp-tls", false, "SMTP TLS")
	flags.String("smtp-user", "", "SMTP Username")
	flags.String("smtp-password", "", "SMTP Password")
	flags.String("smtp-from", "", "SMTP From")
	flags.String("smtp-to", "", "SMTP To : dest of AL")
	flags.String("default-path-to-check", "/ping", "Path to check")
	flags.String("services-path-to-check", "", "Path to check per service : servicea:/path/ServiceA,serviceb:/path/ServiceB")
	flags.String("services-no-http", "", "Services with no HTTP check (only check Sailabove Status) : servicea,serviceb")
	flags.String("services-mail", "", "Services that should use dedicated email alerting instead of Al2tat : servicea,serviceb")
	flags.String("mail-alert-destination", "", "Email address for dedicated email alerting")
	flags.Int("dial-timeout", 2, "dial timeout in seconds")
	flags.Int("read-timeout", 3, "read timeout in seconds")
	flags.Int("dead-line", 4, "deadline in seconds")

	/*not working with ENV var :
	flags.StringSliceVar(&utils.AuthorizedUsers, "authorized-users", nil, "...")*/

	flags.String("authorized-users", "", "Authorized Users, comma separated : firstname.lastname,firstname.lastname")
	flags.String("exclude-services", "", "Exclude some services from tatsamon : serviceA,serviceb")
	flags.String("include-only-services", "", "Include only these services on tatsamon : serviceA,serviceb")

	viper.BindPFlag("production", flags.Lookup("production"))
	viper.BindPFlag("log_level", flags.Lookup("log-level"))
	viper.BindPFlag("listen_port", flags.Lookup("listen-port"))
	viper.BindPFlag("url_al2tat", flags.Lookup("url-al2tat"))
	viper.BindPFlag("url_tat_engine", flags.Lookup("url-tat-engine"))
	viper.BindPFlag("tat_username", flags.Lookup("tat-username"))
	viper.BindPFlag("authorized_users", flags.Lookup("authorized-users"))
	viper.BindPFlag("tat_password", flags.Lookup("tat-password"))
	viper.BindPFlag("tat_alerts_topic", flags.Lookup("tat-alerts-topic"))
	viper.BindPFlag("tat_monitoring_topic", flags.Lookup("tat-monitoring-topic"))
	viper.BindPFlag("no_smtp", flags.Lookup("no-smtp"))
	viper.BindPFlag("smtp_host", flags.Lookup("smtp-host"))
	viper.BindPFlag("smtp_port", flags.Lookup("smtp-port"))
	viper.BindPFlag("smtp_tls", flags.Lookup("smtp-tls"))
	viper.BindPFlag("smtp_user", flags.Lookup("smtp-user"))
	viper.BindPFlag("smtp_password", flags.Lookup("smtp-password"))
	viper.BindPFlag("smtp_from", flags.Lookup("smtp-from"))
	viper.BindPFlag("smtp_to", flags.Lookup("smtp-to"))
	viper.BindPFlag("alert_service", flags.Lookup("alert-service"))
	viper.BindPFlag("exclude_services", flags.Lookup("exclude-services"))
	viper.BindPFlag("include_only_services", flags.Lookup("include-only-services"))
	viper.BindPFlag("activate_cron", flags.Lookup("activate-cron"))
	viper.BindPFlag("sailabove_auth", flags.Lookup("sailabove-auth"))
	viper.BindPFlag("sailabove_host", flags.Lookup("sailabove-host"))
	viper.BindPFlag("default_path_to_check", flags.Lookup("default-path-to-check"))
	viper.BindPFlag("services_path_to_check", flags.Lookup("services-path-to-check"))
	viper.BindPFlag("services_no_http", flags.Lookup("services-no-http"))
	viper.BindPFlag("services_mail", flags.Lookup("services-mail"))
	viper.BindPFlag("mail_alert_destination", flags.Lookup("mail-alert-destination"))
	viper.BindPFlag("activate_alerting", flags.Lookup("activate-alerting"))
	viper.BindPFlag("activate_monitoring", flags.Lookup("activate-monitoring"))
	viper.BindPFlag("cron_check_sailabove", flags.Lookup("cron-check-sailabove"))
	viper.BindPFlag("cron_check_containers", flags.Lookup("cron-check-containers"))
	viper.BindPFlag("enabled_api_applications", flags.Lookup("enabled-api-applications"))

	viper.BindPFlag("dial_timeout", flags.Lookup("dial-timeout"))
	viper.BindPFlag("read_timeout", flags.Lookup("read-timeout"))
	viper.BindPFlag("dead_line", flags.Lookup("dead-line"))

}

func main() {
	mainCmd.Execute()
}
