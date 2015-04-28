# cronner
`cronner` is a simple tool for some basic stats gathering/monitoring around jobs being executed on hosts. As with all tools, it's meant to improve the experience and not be a magic bullet.

`cronner` supports emitting the command run time, and return code, as statsd metrics. In addition, you can have DogStatsd event emissions by enabling it on the command line.

When emitting a finished event, `cronner` provides the combined stdout and stderr output in the body. If the output is too long, it is truncated.

# Usage
Help output:
```
Usage:
  cronner [OPTIONS] command [arguments]...

Application Options:
  -l, --label=               name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it
  -e, --event                emit a start and end datadog event (false)
  -E, --event-fail           only emit an event on failure (false)
  -F, --log-fail             when a command fails, log its full output (stdout/stderr) to the log directory using the UUID as the filename (false)
      --log-path=            where to place the log files for command output (path for -l/--log-on-fail output) (/var/log/cronner/)
  -L, --log-level=           set the level at which to log at [none|error|info|debug] (error)
  -s, --sensitive            specify whether command output may contain sensitive details, this only avoids it being printed to stderr (false)
  -k, --lock                 lock based on label so that multiple commands with the same label can not run concurrently (false)
  -d, --lock-dir=            the directory where lock files will be placed (/var/lock)
  -N, --namespace=           namespace for statsd emissions, value is prepended to metric name by statsd client (cronner)

Help Options:
  -h, --help                 Show this help message

Arguments:
  command [arguments]
```

Running a command:
```
$ cronner -l sleepytime -- /bin/sleep 10
```

Listening to the statsd emissions looks like this:

```
pagerduty.cron.sleepytime.time:10005.834649|ms
pagerduty.cron.sleepytime.exit_code:0|g
```

It emits a timing metric for how long it took for the command to run, as well as the command's exit code.

Running a command and emitting a start and end event:

```
$ cronner -e -l sleepytime2 -- /bin/sleep 5
```

And the statsd interceptions look like this:

```
_e{35,12}:Cron sleepytime2 starting on rinzler|job starting|k:ab31f2f6-498e-468a-b572-ab990065e8d3|s:cron|t:info
pagerduty.cron.sleepytime2.time:5005.649979|ms
pagerduty.cron.sleepytime2.exit_code:0|g
_e{55,22}:Cron sleepytime2 succeeded in 5.00565 seconds on rinzler|exit code: 0\\noutput:(none)|k:ab31f2f6-498e-468a-b572-ab990065e8d3|s:cron|t:success
```

# Development
## Using gpm
```
$ brew install gpm

$ cd cronner

$ export GOPATH=$(pwd)

# install dependencies
$ gpm

# this should produce a cronner binary
$ go build
```

## Without gpm
With the configuration above, you won't be able to import any packages within the `cronner` repo from within the codebase.
In other words, if a cronner file depends on a packge in a subdirectory, you wouldn't be able to locate it within the import path.
To avoid this, you can skip using the gpm approach and use a more manual approach.

* set up your Go build environment
  * install Go: https://golang.org/doc/install
  * set your GOROOT environment variable (in your .bashrc or .zshrc) to the Golang install directory
  * make your GOPATH:
    * `mkdir ~/go`
  * set the GOPATH environment variable in your shell (as well as in your .bashrc or .zshrc) to the directory
    * `export GOPATH=~/go`
  * clone this repo
    * `mkdir -p $GOPATH/src/github.com/PagerDuty && git clone git@github.com:PagerDuty/cronner.git $GOPATH/src/github.com/PagerDuty/cronner`
