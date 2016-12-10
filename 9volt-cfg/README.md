9volt-cfg
=========
A simple utility for converting, validating and dumping YAML based 9volt configs into etcd.

# Usage
`$ 9volt-cfg [-e || --hosts http://localhost:2379] [-p || --prefix 9volt] [-r || --replace] [-d || --debug] [-v || --version] dir`

# Notes
* Point the utility at some dir, it will search for all .yaml files and look for either alerter or monitor maps inside of them
* Collect all monitor and alerter configurations -> perform validation (dupes, etc.)
* By default, perform checks in etcd to see if an identical config exists and do not replace it
    * This behavior can be changed by supplying the -r flag to replace existing values (ie. do not check if new config value is different than what's in etcd)
* Push converted configs to etcd host(s)
