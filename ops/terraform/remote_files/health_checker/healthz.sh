#!/bin/bash
echo "Status: 200\r\n\r\n"

date --rfc-3339=ns
echo "uptime:"
uptime
echo "Currently connected:"
w
echo "--------------------"
echo "Last logins:"
last -a | head -3
echo "--------------------"
echo "Disk and memory usage:"
df -h | xargs | awk '{print "Free/total disk: " $11 " / " $9}'
free -m | xargs | awk '{print "Free/total memory: " $17 " / " $8 " MB"}'
echo "--------------------"
start_log=$(journalctl | head -1 | cut -c 1-15)
oom=$(journalctl | grep -ci kill)
echo -n "OOM errors since $start_log :" $oom
echo ""
echo "--------------------"
echo "Utilization and most expensive processes:"
ps -Ao user,uid,comm,pid,pcpu,tty --sort=-pcpu | head -n 6
echo "--------------------"
echo "Current connections:"
ss -s
echo "--------------------"
echo "processes:"
ps auxf --width=200
echo "--------------------"
echo "vmstat:"
vmstat 1 5
echo "--------------------"
echo "$(bacalhau --version)"
echo "--------------------"
echo "PATH: $PATH"
echo "--------------------"
echo "$(ps aux | grep -E 'ipfs|bacalhau')"