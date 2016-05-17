#!/usr/bin/env bash
#
# Initial 9volt setup script
#

PREFIX="9volt"

if [ "$#" -ne 1 ]; then
	echo "Usage: ./setup.sh http://some.etcd-host.com:2379"
	exit 1
fi

EXISTS=$(which curl)

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

warningMessage

# Add initial config
curl http://127.0.0.1:2379/v2/keys/$PREFIX/config -XPUT -d value="{\"HeartbeatInterval\":\"5s\",\"HeartbeatTimeout\":\"10s\"}"

# Create initial dirs
curl http://127.0.0.1:2379/v2/keys/$PREFIX/alert -XPUT -d dir=true
curl http://127.0.0.1:2379/v2/keys/$PREFIX/host -XPUT -d dir=true
curl http://127.0.0.1:2379/v2/keys/$PREFIX/monitor -XPUT -d dir=true
curl http://127.0.0.1:2379/v2/keys/$PREFIX/cluster -XPUT -d dir=true
curl http://127.0.0.1:2379/v2/keys/$PREFIX/cluster/members -XPUT -d dir=true