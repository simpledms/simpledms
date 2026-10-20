# Runtime profiling

Set `SIMPLEDMS_PPROF_ENABLED=true` and restart/recreate the app. This also works in
the SaaS wrapper. The setting is read at startup independently of database
configuration overrides. It defaults to disabled.

Profiling is served at `/debug/pprof/` on the existing application origin and port,
including in Docker. Sign in with a **system administrator** account using a full
session; ordinary users, tenant administrators, and temporary/setup sessions
cannot access profiles. The regular application session checks also apply here.

## Collect an idle CPU profile

While signed in, open this URL in your browser, replacing the hostname:

```text
https://YOUR_APP_HOST/debug/pprof/profile?seconds=120
```

Leave the app idle during the two-minute capture and save the download as
`idle-cpu.pprof`. The index at `/debug/pprof/` lists the available profiles.
For scripted collection, use an exported cookie jar for your system-admin session:

```sh
curl --fail --show-error --cookie admin-cookies.txt --output idle-cpu.pprof 'https://YOUR_APP_HOST/debug/pprof/profile?seconds=120'
go tool pprof -top idle-cpu.pprof
go tool pprof -top -cum idle-cpu.pprof
```

The cookie jar must contain a valid session for that hostname. An expired session
redirects to sign-in rather than producing a profile. If your reverse proxy has a
shorter response timeout, increase it for `/debug/pprof/` or collect a shorter
profile, for example `seconds=30`.

Use `go tool pprof -http=127.0.0.1:8081 idle-cpu.pprof` for an interactive local view.
Keep the deployed binary for symbolization if needed:
`go tool pprof -top ./simpledms idle-cpu.pprof`.

## Other diagnostics

These URLs use the same signed-in session:

```text
/debug/pprof/heap
/debug/pprof/goroutine?debug=2
/debug/pprof/trace?seconds=5
```

Save trace output as `trace.out` and open it with `go tool trace trace.out`.
CPU profiling and tracing run only when requested; block and mutex sampling remain
at their Go defaults. Responses are marked `Cache-Control: no-store`.

For the idle-load investigation, collect the CPU profile first and record process
CPU percentage, host load average, and whether background conversions/backfills
were running. CPU profiles measure execution time, not time blocked on disk I/O.

Unset `SIMPLEDMS_PPROF_ENABLED` or set it to `false` and restart to disable access.
