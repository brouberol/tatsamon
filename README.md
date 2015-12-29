# Overview

Tatsamon monitors your Sailabove containers.

Every 10min : list all yours containers,

* If not runnning -> send an alarm and/or event to Tat through the al2tat (https://github.com/ovh/al2tat) service :
Every 30s : HTTP GET on a path on all your container (default path : ``/ping``, flag path_to_check).

* If not OK (no response or http status not 200) -> send an alarm and/or event to Tat (https://github.com/ovh/tat) through the al2tat service :
Tatsamon has to be deployed with your sailabove account. Please ask Team CD to deploy an instance for you.

# Rules

One alarm and/or event max per 10min sent to Tat (https://github.com/ovh/tat)
If service tat or al2tat is down -> send a mail

# Alarm and/or event
AL2Tat manages alerts and event monitoring.

Tatsamon can generate alerts and or event monitoring on Tat.

* An alert is purged (about 60 days after creation on tat). Best Tat view for theses : RunView

* An event monitoring is attached to an item (host, soft, person... whatever), all events are held three days, and only 30 events are retained after 3 days. Best Tat view for theses : Monitoring View

You can activate both with tatsamon.

# Usage

## Scheduler

Tatsamon can be used by two ways : internal scheduler or external sheduler.

* Internal scheduler : every 10 minutes (flag cron-check-sailabove), call sailabove API and check all containers every 30 seconds (flag cron-check-containers)
* External scheduler : you have to use an external system to make a HTTP GET on ``/containers/check``. Call Sailabove API every 10 minutes (flag cron-check-sailabove) to refresh the list of containers.

## API

### ``GET /applications``

This endpoint gives you sailabove information on your containers and refreshes the internal tatsamon list of containers.

### ``GET /containers/check``

This endpoint can be called by an external scheduler to run test on your containers.

## Options
```
Flags:
      --activate-alerting[=true]: Activate Alerts Generation to Al2tat service
      --activate-cron[=true]: Activate internal tatsamon cron
      --activate-monitoring[=true]: Activate Event Monitoring to Al2tat service
      --alert-service="": Your Service for al2tat message
      --authorized-users="": Authorized Users, comma separated : firstname.lastname,firstname.lastname
      --cron-check-containers=30: If activate-cron=true, seconds before each call each containers
      --cron-check-sailabove=600: If activate-cron=true, seconds before each call to API Sailabove
      --dead-line=3: deadline in seconds
      --default-path-to-check="/ping": Path to check
      --dial-timeout=1: dial timeout in seconds
      --exclude-services="": Exclude some services from tatsamon : serviceA,serviceb
  -h, --help[=false]: help for tatsamon
      --include-only-services="": Include only these services on tatsamon : serviceA,serviceb
      --listen-port="8085": Tatmon Listen Port
      --log-level="": Log Level : debug, info or warn
      --no-smtp[=false]: No SMTP mode
      --production[=false]: Production mode
      --read-timeout=2: read timeout in seconds
      --sailabove-auth="": Sailabove Auth (base64)
      --sailabove-host="sailabove.io": Sailabove Host
      --services-path-to-check="/ping": Path to check per service : servicea:/path/ServiceA,serviceb:/path/ServiceB
      --smtp-from="": SMTP From
      --smtp-host="": SMTP Host
      --smtp-password="": SMTP Password
      --smtp-port="": SMTP Port
      --smtp-tls[=false]: SMTP TLS
      --smtp-to="": SMTP To : dest of AL
      --smtp-user="": SMTP Username
      --tat-alerts-topic="/Internal/Alerts": Tat Alerts Topic
      --tat-monitoring-topic="": Tat Monitoring Topic
      --tat-password="": Tat Password
      --tat-username="tat.system.tatsamon": Tat Username
      --url-al2tat="http://localhost:8082": URL AL2Tat
      --url-tat-engine="http://localhost:8080": URL Tat Engine

```

## Viewing on Tatwebui

You can view sailabove monitoring with the monitoring view (https://github.com/ovh/tatwebui-plugin-monitoringview) on
your Tatwebui (https://github.com/ovh/tatwebui) installation.

# Hacking

Tatsamon is written in Go 1.5, using the experimental vendoring
mechanism introduced in this version. Make sure you are using at least
version 1.5.

```bash
mkdir -p $GOPATH/src/github.com/ovh
cd $GOPATH/src/github.com/ovh
git clone git@github.com:ovh/tatsamon.git
cd $GOPATH/src/github.com/ovh/tatsamon
export GO15VENDOREXPERIMENT=1
go build
```

You've developed a new cool feature? Fixed an annoying bug? We'd be happy
to hear from you! Make sure to read [CONTRIBUTING.md](./CONTRIBUTING.md) before.
