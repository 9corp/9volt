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
  # Add initial config
  curl -s $ETCDHOST/v2/keys/$PREFIX/config -XPUT -d value="{\"HeartbeatInterval\":\"3s\",\"HeartbeatTimeout\":\"6s\"}"

  # Create initial dirs
  curl -s $ETCDHOST/v2/keys/$PREFIX/alert -XPUT -d dir=true
  curl -s $ETCDHOST/v2/keys/$PREFIX/host -XPUT -d dir=true
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor -XPUT -d dir=true
  curl -s $ETCDHOST/v2/keys/$PREFIX/cluster -XPUT -d dir=true
  curl -s $ETCDHOST/v2/keys/$PREFIX/cluster/members -XPUT -d dir=true
}

createSampleChecks() {
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/some_config_1 -XPUT -d value="{\"stuff\" : 1}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/some_config_2 -XPUT -d value="{\"stuff\" : 2}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/some_config_3 -XPUT -d value="{\"stuff\" : 3}"
  curl -s $ETCDHOST/v2/keys/$PREFIX/monitor/some_config_4 -XPUT -d value="{\"stuff\" : 4}"
}

warningMessage
setupEtcd
createSampleChecks
