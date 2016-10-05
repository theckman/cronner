# cronner
[![TravisCI Build Status](https://img.shields.io/travis/PagerDuty/cronner/master.svg?style=flat)](https://travis-ci.org/PagerDuty/cronner)

`cronner` is a command line utility to that wraps periodic (cron) jobs for statistics gathering and success monitoring. The amount of time the command took to ran, as well as the return code, are emitted as vanilla statsd metrics to port 8125. It also implements file-level locking for very simple, and dumb, job semaphore.

The utility also supports emitting [DogStatsD Events](http://docs.datadoghq.com/guides/dogstatsd/#events) under the following occasions:

* job start and job finish
* job finish if the job failed
* if the job is taking too long to finish running

If your statsd agent isn't DogStatsD-compliant, I'm not sure what the behavior will be if you an emit an event to it.

For the finish DogStatsD event, the return code and output of the command are provided in the event body. If the output is too long, it is truncated. This output can optionally be saved to disk only if the job fails for later inspection.

## License
Cronner is released under the BSD 3-Clause License. See the `LICENSE` file for
the full contents of the license.

## Usage
### Help Output

```
Usage:
  cronner [OPTIONS] -- command [arguments]...

Application Options:
  -d, --lock-dir=                  the directory where lock files will be placed (/var/lock)
  -e, --event                      emit a start and end datadog event (false)
  -E, --event-fail                 only emit an event on failure (false)
  -F, --log-fail                   when a command fails, log its full output (stdout/stderr) to the log directory using the UUID as the filename (false)
  -G, --event-group=<group>        emit a cronner_group:<group> tag with Datadog events, does not get sent with statsd metrics
  -k, --lock                       lock based on label so that multiple commands with the same label can not run concurrently (false)
  -l, --label=                     name for cron job to be used in statsd emissions and DogStatsd events. alphanumeric only; cronner will lowercase it
      --log-path=                  where to place the log files for command output (path for -F/--log-fail output) (/var/log/cronner)
  -L, --log-level=                 set the level at which to log at [none|error|info|debug] (error)
  -N, --namespace=                 namespace for statsd emissions, value is prepended to metric name by statsd client (cronner)
  -p, --passthru                   passthru stdout/stderr to controlling tty
  -s, --sensitive                  specify whether command output may contain sensitive details, this only avoids it being printed to stderr (false)
  -V, --version                    print the version string and exit
  -w, --warn-after=N               emit a warning event every N seconds if the job hasn't finished, set to 0 to disable (0)
  -W, --wait-secs=                 how long to wait for the file lock for (0)

Help Options:
  -h, --help                       Show this help message

Arguments:
  -- command [arguments]
```

### Running A Command
The label (`-l`, `--label`) flag is required.

To run the command `/bin/sleep 10` and emit the stats as `cronner.sleeptyime.time` and `cronner.sleepytime.exit_code` you would run:

```
$ cronner -l sleepytime -- /bin/sleep 10
```

To note, `--` in the command line arguments tells cronner to stop parsing CLi flags. It then grabs the rest of the arguments as the command to execute.

If you were to have a UDP listener on port 8125 on localhost, the statsd emissions would look something like this:

```
cronner.sleepytime.time:10005.834649|ms
cronner.sleepytime.exit_code:0|g
```

It emits a timing metric for how long it took for the command to run, as well as the command's exit code.

### Running A Command with a DogStatsD Event
If you want to run `/bin/sleep 5` as `sleepytime2` and emit a DogStatsD for when the job starts and finishes:

```
$ cronner -e -l sleepytime2 -- /bin/sleep 5
```

The UDP datagrams emitted would then look like this:

```
_e{35,12}:Cron sleepytime2 starting on rinzler|job starting|k:ab31f2f6-498e-468a-b572-ab990065e8d3|s:cronner|t:info
cronner.sleepytime2.time:5005.649979|ms
cronner.sleepytime2.exit_code:0|g
_e{55,22}:Cron sleepytime2 succeeded in 5.00565 seconds on rinzler|exit code: 0\\noutput:(none)|k:ab31f2f6-498e-468a-b572-ab990065e8d3|s:cronner|t:success
```

## Contributors
* Tim Heckman
* Thomas Dziedzic

## Development
* set up your workspace as per the instructions for standard Go development
* clone the cronner repository

  ```BASH
  git clone git@github.com:PagerDuty/cronner.git
  ```
* make your changes to the codebase, including adding relevant test cases
* run your tests to ensure all pass

  ```BASH
  go test -v ./... -check.vv
  ```
* confirm that building cronner works

  ```BASH
  go build
  ```
