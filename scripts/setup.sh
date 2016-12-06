#!/usr/bin/env bash
#
# Initial 9volt setup script
#

[ -z "$PREFIX" ] && PREFIX="9volt"
[ -z "$ETCDHOST" ] && ETCDHOST="http://127.0.0.1:2379"

EXISTS=$(hash curl)

if [ $? != "0" ]; then
	echo "ERROR: Curl does not appear to be available"
	exit 1
fi

warningMessage() {
	echo "+--------------------------------------------------------------+"
	echo "|            !!! This is a destructive operation !!!           |"
	echo "|                                                              |"
	echo "|  Running this script will wipe out all 9volt configuration!  |"
	echo "|                                                              |"
	echo "|          Press [ENTER] to continue or CTRL-C to quit         |"
	echo "+--------------------------------------------------------------+"
	read
}

setupEtcd() {
  # Delete any initial configuration
  curl -s $ETCDHOST/v2/keys/$PREFIX?recursive=true -X DELETE

  # Add initial config
  curl -s $ETCDHOST/v2/keys/$PREFIX/config -XPUT -d value="{\"HeartbeatInterval\":\"3s\",\"HeartbeatTimeout\":\"6s\"}"

  # Create initial dirs
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter -XPUT -d dir=true
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor -XPUT -d dir=true
  curl -s $ETCDHOST/v2/keys/$PREFIX/cluster -XPUT -d dir=true
  curl -s $ETCDHOST/v2/keys/$PREFIX/cluster/members -XPUT -d dir=true
}

createSampleMonitorConfigs() {
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/monitor_config_1 -XPUT -d value="{\"type\":\"http\",\"description\":\"example monitor config 1\",\"timeout\":\"5s\",\"interval\":\"4s\"}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/monitor_config_2 -XPUT -d value="{\"type\":\"http\",\"description\":\"example monitor config 2\",\"timeout\":\"5s\",\"interval\":\"6s\"}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/monitor_config_3 -XPUT -d value="{\"type\":\"http\",\"description\":\"example monitor config 3\",\"timeout\":\"5s\",\"interval\":\"8s\"}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/monitor_config_4 -XPUT -d value="{\"type\":\"http\",\"description\":\"example monitor config 4\",\"timeout\":\"5s\",\"interval\":\"10s\"}"
}

createSampleAlerterConfigs() {
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/pd_config_1 -XPUT -d value="{\"type\":\"pagerduty\",\"description\":\"foobar1-pd\",\"options\":{\"apikey\":\"api-key-1\",\"custom-key\":\"custom-data-1\"}}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/pd_config_2 -XPUT -d value="{\"type\":\"pagerduty\",\"description\":\"foobar2-pd\",\"options\":{\"apikey\":\"api-key-2\",\"custom-key\":\"custom-data-2\"}}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/sl_config_1 -XPUT -d value="{\"type\":\"slack\",\"description\":\"foobar1-sl\",\"options\":{\"apikey\":\"api-key-1\",\"custom-key\":\"custom-data-1\"}}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/sl_config_2 -XPUT -d value="{\"type\":\"slack\",\"description\":\"foobar2-sl\",\"options\":{\"apikey\":\"api-key-2\",\"custom-key\":\"custom-data-2\"}}" 
}

warningMessage
setupEtcd
createSampleMonitorConfigs
createSampleAlerterConfigs
