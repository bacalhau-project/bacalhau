$logfile = $Env:LOTUS_LOGFILE.split("\")
$logfile[0] = "/" + $logfile[0].replace(":", "").toLower()
$ENV:LOTUS_LOGFILE = $logfile -join "/"

sh .\lotus.sh $args