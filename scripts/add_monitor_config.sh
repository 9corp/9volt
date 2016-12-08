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
	echo "|          Press [ENTER] to continue or CTRL-C to quit         |"
	echo "+--------------------------------------------------------------+"
	read
}

wipe() {
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/?recursive=true -X DELETE
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/?recursive=true -X DELETE
}

createSampleMonitorConfigs() {
#  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/monitor_config_1 -XPUT -d value="{\"Type\":\"http\",\"Description\":\"200 http status check\",\"Host\":\"localhost\",\"Timeout\":\"5s\",\"Interval\":\"10s\",\"Enabled\":true,\"Method\":\"GET\",\"Port\":8080,\"SSL\":false,\"URL\":\"/healthcheck\",\"StatusCode\":200,\"Tags\":[\"team-core\",\"golang\"],\"WarningThreshold\":1,\"CriticalThreshold\":3,\"WarningAlerter\":[\"my-email\",\"my-slack\"],\"CriticalAlerter\":[\"my-slack\",\"my-pagerduty\"]}"
#  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/monitor_config_2 -XPUT -d value="{\"Type\":\"http\",\"Description\":\"200 http status check\",\"Host\":\"localhost\",\"Timeout\":\"5s\",\"Interval\":\"1m\",\"Enabled\":true,\"Method\":\"GET\",\"Port\":8181,\"SSL\":false,\"URL\":\"/healthcheck\",\"StatusCode\":200,\"Tags\":[\"team-core\",\"golang\"],\"WarningThreshold\":1,\"CriticalThreshold\":3,\"WarningAlerter\":[\"other-email\",\"other-slack\"],\"CriticalAlerter\":[\"other-slack\",\"other-pagerduty\"]}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/cloudsy_tcp_check -XPUT -d value="{\"Type\":\"tcp\",\"Description\":\"remote tcp check, no thrills\",\"Host\":\"cloudsy.com\",\"Timeout\":\"5s\",\"Interval\":\"10s\",\"Enabled\":true,\"Port\":12345,\"Tags\":[\"team-core\",\"golang\"],\"WarningThreshold\":1,\"CriticalThreshold\":3,\"WarningAlerter\":[\"other-email\",\"other-slack\"],\"CriticalAlerter\":[\"other-slack\",\"other-pagerduty\"]}"
# curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/cloudsy_tcp_check_expect -XPUT -d value="{\"Type\":\"tcp\",\"Description\":\"remote tcp check, expect string\",\"Host\":\"cloudsy.com\",\"Timeout\":\"5s\",\"Interval\":\"10s\",\"Enabled\":true,\"Port\":22,\"Expect\":\"OpenSSH\",\"Tags\":[\"team-core\",\"golang\"],\"WarningThreshold\":1,\"CriticalThreshold\":3,\"WarningAlerter\":[\"other-email\",\"other-slack\"],\"CriticalAlerter\":[\"other-slack\",\"other-pagerduty\"]}"
#  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/cloudsy_tcp_check_low_read_timeout -XPUT -d value="{\"Type\":\"tcp\",\"Description\":\"remote tcp check, really low read timeout\",\"Host\":\"cloudsy.com\",\"Timeout\":\"5s\",\"ReadTimeout\":\"1ms\",\"Interval\":\"10s\",\"Enabled\":true,\"Port\":22,\"Expect\":\"OpenSSH\",\"Tags\":[\"team-core\",\"golang\"],\"WarningThreshold\":1,\"CriticalThreshold\":3,\"WarningAlerter\":[\"other-email\",\"other-slack\"],\"CriticalAlerter\":[\"other-slack\",\"other-pagerduty\"]}"
#  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/cloudsy_tcp_check_send_first -XPUT -d value="{\"Type\":\"tcp\",\"Description\":\"remote tcp check, send first, then expect\",\"Host\":\"cloudsy.com\",\"Timeout\":\"5s\",\"Interval\":\"10s\",\"Enabled\":true,\"Port\":22,\"Send\":\"Hello\\n\",\"Expect\":\"OpenSSH\",\"Tags\":[\"team-core\",\"golang\"],\"WarningThreshold\":1,\"CriticalThreshold\":3,\"WarningAlerter\":[\"other-email\",\"other-slack\"],\"CriticalAlerter\":[\"other-slack\",\"other-pagerduty\"]}"
}

createSampleAlerterConfigs() {
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/my-pagerduty -XPUT -d value="{\"type\":\"pagerduty\",\"description\":\"my pagerduty alerter\",\"options\":{\"apikey\":\"1234567890\",\"custom-key\":\"custom data used by the pagerduty alerter\"}}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/other-pagerduty -XPUT -d value="{\"type\":\"pagerduty\",\"description\":\"other pagerduty alerter\",\"options\":{\"apikey\":\"1234567890\",\"custom-key\":\"custom data used by the pagerduty alerter\"}}" 
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/my-slack -XPUT -d value="{\"type\":\"slack\",\"description\":\"my slack alerter\",\"options\":{\"token\":\"foo\",\"channel\":\"9volt-testing\",\"username\":\"robibi\",\"iconURL\":\"http://cdn.akamai.steamstatic.com/steamcommunity/public/images/avatars/d2/d25fd479e446f3bef884cbedb5b2b643133b93fc_full.jpg\"}}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/other-slack -XPUT -d value="{\"type\":\"slack\",\"description\":\"other slack alerter\",\"options\":{\"token\":\"bar\",\"channel\":\"9volt-testing\"}}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/my-email -XPUT -d value="{\"type\":\"email\",\"description\":\"my email alerter\",\"options\":{\"email\":\"daniel.selans@gmail.com\"}}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/alerter/other-email -XPUT -d value="{\"type\":\"email\",\"description\":\"other email alerter\",\"options\":{\"email\":\"daniel.selans@gmail.com\"}}"
}

warningMessage
wipe
createSampleMonitorConfigs
createSampleAlerterConfigs
