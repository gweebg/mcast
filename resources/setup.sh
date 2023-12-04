#!/bin/bash
set +x

#setup_script="cd /home/core/mcast; export DISPLAY=:0.0; export TERM=xterm"

pycore="/tmp/pycore.44251"
confs=$(ls "$pycore" | grep '^[[:upper:]].*.conf')
nodes=$(ls "$pycore" | grep '^[[:upper:]].*.conf' | awk -F '.' '{print $1}')
clients=$(echo "$nodes" | grep 'Client')
Os=$(echo "$nodes" | grep 'O')
servers=$(echo "$nodes" | grep 'Server')
RP=$(echo "$nodes" | grep 'RP')
boot=$(echo "$nodes" | grep 'Bootstrapper')

term="xfce4-terminal "
command=$term
i=0
for node in $clients; do
    if [[ $i -gt 0 ]]; then
        command+="--tab -T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    else
        command+="-T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    fi
    i+=1
done
eval "$command"

command=$term
i=0
for node in $servers; do
    if [[ $i -gt 0 ]]; then
        command+="--tab -T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    else
        command+="-T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    fi
    i+=1
done
eval "$command"

command=$term
i=0
for node in $RP; do
    if [[ $i -gt 0 ]]; then
        command+="--tab -T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    else
        command+="-T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    fi
    i+=1
done
eval "$command"

command=$term
i=0
for node in $Os; do
    if [[ $i -gt 0 ]]; then
        command+="--tab -T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    else
        command+="-T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    fi
    i+=1
done
eval "$command"

command=$term
i=0
for node in $boot; do
    if [[ $i -gt 0 ]]; then
        command+="--tab -T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    else
        command+="-T $node -e \"sh -c 'vcmd -c $pycore/$node'\" "
    fi
    i+=1
done
eval "$command"
