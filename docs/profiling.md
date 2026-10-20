# Runtime profiling

Set `SIMPLEDMS_PPROF_PATH` to a random secret URL segment and restart/recreate the
app. For example, generate a value with `openssl rand -hex 24` and configure:

```env
SIMPLEDMS_PPROF_PATH=YOUR_RANDOM_VALUE
```

This also works in the SaaS wrapper. The setting is read at startup independently
of database configuration overrides. An empty value disables access. Only ASCII
letters, digits, `-`, and `_` are accepted; invalid values log an error and leave
profiling disabled. The former `SIMPLEDMS_PPROF_ENABLED` setting is no longer used.

Profiling is served at `/<configured-value>/debug/pprof/` on the existing origin and port,
including in Docker. Sign in with a **system administrator** account using a full
session; ordinary users, tenant administrators, and temporary/setup sessions
cannot access profiles. The regular application session checks also apply here.

## Collect an idle CPU profile

While signed in, open this URL in your browser, replacing the hostname and secret:

```text
https://YOUR_APP_HOST/YOUR_RANDOM_VALUE/debug/pprof/profile?seconds=120
```

Leave the app idle during the two-minute capture and save the download as
`idle-cpu.pprof`. The index at `/YOUR_RANDOM_VALUE/debug/pprof/` lists the profiles.
For scripted collection, use an exported cookie jar for your system-admin session:

```sh
curl --fail --show-error --cookie admin-cookies.txt --output idle-cpu.pprof 'https://YOUR_APP_HOST/YOUR_RANDOM_VALUE/debug/pprof/profile?seconds=120'
go tool pprof -top idle-cpu.pprof
go tool pprof -top -cum idle-cpu.pprof
```

The cookie jar must contain a valid session for that hostname. An expired session
redirects to sign-in rather than producing a profile. If your reverse proxy has a
shorter response timeout, increase it for the configured subtree or collect a shorter
profile, for example `seconds=30`.

Use `go tool pprof -http=127.0.0.1:8081 idle-cpu.pprof` for an interactive local view.
Keep the deployed binary for symbolization if needed:
`go tool pprof -top ./simpledms idle-cpu.pprof`.

## Other diagnostics

These URLs use the same signed-in session:

```text
/YOUR_RANDOM_VALUE/debug/pprof/heap
/YOUR_RANDOM_VALUE/debug/pprof/goroutine?debug=2
/YOUR_RANDOM_VALUE/debug/pprof/trace?seconds=5
```

Save trace output as `trace.out` and open it with `go tool trace trace.out`.
CPU profiling and tracing run only when requested; block and mutex sampling remain
at their Go defaults. Responses are marked `Cache-Control: no-store` and
`Referrer-Policy: no-referrer`.

For the idle-load investigation, collect the CPU profile first and record process
CPU percentage, host load average, and whether background conversions/backfills
were running. CPU profiles measure execution time, not time blocked on disk I/O.

Unset `SIMPLEDMS_PPROF_PATH` or set it to an empty value and restart to disable
access. Changing its value and restarting rotates the URL. The unprefixed
`/debug/pprof/` URL is not registered.
