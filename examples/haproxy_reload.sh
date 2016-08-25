#!/usr/bin/env bash
set -x

PID_FILE=/tmp/haproxy.pid

nl-qdisc-add --dev=lo --parent=1:4 --id=40: --update plug --buffer
/usr/sbin/haproxy -f ${HAP_CONFIG} -p ${PID_FILE} -D  $([ -f ${PID_FILE} ] && echo -sf $(cat ${PID_FILE}))
fail=$?
nl-qdisc-add --dev=lo --parent=1:4 --id=40: --update plug --release-indefinit

exit 1