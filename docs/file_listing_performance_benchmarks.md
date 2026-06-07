# File Listing Performance Benchmarks

Benchmarks were run on Sun May 31 2026 with:

```sh
go test -run '^$' -bench '^BenchmarkFileListing$/^(inbox_1000|browse_1000|inbox_10000|browse_10000|inbox_100000|browse_100000)$/^(query|handler|router)$' -benchtime=1x -count=1 ./server
```

Environment reported by Go:

```text
goos: linux
goarch: amd64
pkg: github.com/simpledms/simpledms/server
cpu: AMD Ryzen 9 5950X 16-Core Processor
```

## Baseline

| Benchmark | ns/op | ms/event | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| inbox_1000/query | 739462 | 0.7395 | 54256 | 1070 |
| inbox_1000/handler | 2172377 | 2.172 | 453024 | 5397 |
| inbox_1000/router | 3003741 | 3.004 | 518952 | 5736 |
| browse_1000/query | 656191 | 0.6562 | 173536 | 3111 |
| browse_1000/handler | 23797058 | 23.80 | 9555712 | 128056 |
| browse_1000/router | 25785531 | 25.79 | 9696736 | 128969 |
| inbox_10000/query | 4580068 | 4.580 | 53552 | 1068 |
| inbox_10000/handler | 5347561 | 5.348 | 406824 | 4736 |
| inbox_10000/router | 6964269 | 6.964 | 566544 | 5771 |
| browse_10000/query | 2542743 | 2.543 | 174896 | 3113 |
| browse_10000/handler | 25668530 | 25.67 | 9564096 | 127910 |
| browse_10000/router | 26269351 | 26.27 | 9648280 | 128947 |
| inbox_100000/query | 43817261 | 43.82 | 53552 | 1068 |
| inbox_100000/handler | 44544377 | 44.54 | 370376 | 4737 |
| inbox_100000/router | 45687720 | 45.69 | 562152 | 5756 |
| browse_100000/query | 19033567 | 19.03 | 175200 | 3116 |
| browse_100000/handler | 38753044 | 38.75 | 9509536 | 127865 |
| browse_100000/router | 41781404 | 41.78 | 9673768 | 128899 |

## After Optimization (Initial 1x Run)

| Benchmark | ns/op | ms/event | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| inbox_1000/query | 442167 | 0.4422 | 73376 | 520 |
| inbox_1000/handler | 1429903 | 1.430 | 390696 | 4828 |
| inbox_1000/router | 2164874 | 2.165 | 496760 | 5190 |
| browse_1000/query | 531448 | 0.5314 | 124168 | 2436 |
| browse_1000/handler | 25458505 | 25.46 | 9497448 | 127431 |
| browse_1000/router | 24564802 | 24.56 | 9643376 | 128284 |
| inbox_10000/query | 338786 | 0.3388 | 73408 | 522 |
| inbox_10000/handler | 1274590 | 1.275 | 384296 | 4184 |
| inbox_10000/router | 2254595 | 2.255 | 541872 | 5213 |
| browse_10000/query | 2148924 | 2.149 | 124280 | 2440 |
| browse_10000/handler | 25730280 | 25.73 | 9490264 | 127293 |
| browse_10000/router | 28158738 | 28.16 | 9626096 | 128343 |
| inbox_100000/query | 439377 | 0.4394 | 73632 | 524 |
| inbox_100000/handler | 1320200 | 1.320 | 384296 | 4184 |
| inbox_100000/router | 2435208 | 2.435 | 491528 | 5191 |
| browse_100000/query | 16417906 | 16.42 | 124280 | 2440 |
| browse_100000/handler | 40029372 | 40.03 | 9475448 | 127270 |
| browse_100000/router | 42303407 | 42.30 | 9661224 | 128284 |

## Comparison

| Benchmark | Baseline ms/event | Optimized ms/event | Change |
| --- | ---: | ---: | ---: |
| inbox_1000/query | 0.7395 | 0.4422 | 40.2% faster |
| inbox_1000/handler | 2.172 | 1.430 | 34.2% faster |
| inbox_1000/router | 3.004 | 2.165 | 27.9% faster |
| browse_1000/query | 0.6562 | 0.5314 | 19.0% faster |
| browse_1000/handler | 23.80 | 25.46 | 7.0% slower |
| browse_1000/router | 25.79 | 24.56 | 4.8% faster |
| inbox_10000/query | 4.580 | 0.3388 | 92.6% faster |
| inbox_10000/handler | 5.348 | 1.275 | 76.2% faster |
| inbox_10000/router | 6.964 | 2.255 | 67.6% faster |
| browse_10000/query | 2.543 | 2.149 | 15.5% faster |
| browse_10000/handler | 25.67 | 25.73 | 0.2% slower |
| browse_10000/router | 26.27 | 28.16 | 7.2% slower |
| inbox_100000/query | 43.82 | 0.4394 | 99.0% faster |
| inbox_100000/handler | 44.54 | 1.320 | 97.0% faster |
| inbox_100000/router | 45.69 | 2.435 | 94.7% faster |
| browse_100000/query | 19.03 | 16.42 | 13.7% faster |
| browse_100000/handler | 38.75 | 40.03 | 3.3% slower |
| browse_100000/router | 41.78 | 42.30 | 1.2% slower |

Key observations:

- Inbox search/listing now stays around 1.3-2.4 ms through the handler/router path at 100k files, instead of growing to roughly 45 ms.
- In the initial 1x run, browse query time improved 14-19% in the measured sizes, mostly from avoiding eager child-row loading and using better indexes.
- Browse handler/router timings are still dominated by rendering the 50-row list/table fragment and are noisy at `-benchtime=1x`; the database query itself improved.
