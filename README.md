# cronner
`cronner` is a simple tool for some basic stats gathering/monitoring around jobs being executed on hosts. As with all tools, it's meant to improve the experience and not be a magic bullet.

`cronner` supports emitting the command run time, and return code, as statsd metrics. In addition, you can have DogStatsd event emissions by enabling it on the command line.

When emitting a finished event, `cronner` provides the combined stdout and stderr output in the body. If the output is too long, it is truncated.

# Usage
Help output:
```
Usage:
  cronner [OPTIONS]

Application Options:
  -l, --label=   name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it
  -c, --command= command to run (please use full path) and its args; executed as user running cronner
  -e, --event    emit a start and end datadog event (false)

Help Options:
  -h, --help     Show this help message
```

Running a command:
```
$ cronner -c 'sleep 10' -l sleepytime
```

Listening to the statsd emissions looks like this:

```
cron.sleepytime.time:10005.834649|ms
cron.sleepytime.exit_code:0|g
```

It emits a timing metric for how long it took for the command to run, as well as the command's exit code.

Running a command and emitting a start and end event:

```
./cronner -e -l sleepytime2 -c 'sleep 5'
```

And the statsd interceptions look like this:

```
_e{35,12}:Cron sleepytime2 starting on rinzler|job starting|k:ab31f2f6-498e-468a-b572-ab990065e8d3|s:cron|t:info
cron.sleepytime2.time:5005.649979|ms
cron.sleepytime2.exit_code:0|g
_e{55,22}:Cron sleepytime2 succeeded in 5.00565 seconds on rinzler|exit code: 0\\noutput:|k:ab31f2f6-498e-468a-b572-ab990065e8d3|s:cron|t:success
```
