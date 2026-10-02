# livestats

单场直播观看统计库（纯进程内存，无外部依赖）。

## 用法

```go
s := livestats.New()
s.Report("viewer-a", livestats.Segment{Start: 0, End: 10_000})
st := s.Query(0, 20_000) // st.Viewers / TotalWatchMs / AvgWatchMs / PeakConcurrent
```

- 时间为相对开播的整数毫秒，片段与查询窗口均为 `[start, end)`。
- 同一观众的重叠片段只计一份时长、占一个在线名额；不同观众分别计数。
- 片段可乱序、重复、迟到上报；上报幂等，只补充事实。
- 查询返回窗口内观看人数、有效观看时长总和、平均观看时长、峰值同时在线人数。

## 测试

```sh
go test ./...
```
