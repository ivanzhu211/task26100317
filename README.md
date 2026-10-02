# livestat

单场直播观看统计库。观看记录与统计结果保存在进程内存中，由上层直接调用。

## 用法

```go
tr := livestat.NewTracker()

// 上报已完成配对的观看片段 [start, end)，相对开播的毫秒数。
// 片段可乱序、重复或迟到；重复上报不改变统计结果。
tr.Report("viewer-a", 0, 10_000)
tr.Report("viewer-a", 5_000, 15_000)
tr.Report("viewer-b", 15_000, 20_000)

// 查询任意窗口 [start, end) 的统计。
s := tr.Query(0, 20_000)
fmt.Println(s.Viewers, s.TotalMillis, s.AvgMillis(), s.PeakConcurrent)
```

## 语义

- 片段与查询窗口均为半开区间 `[start, end)`，只统计已上报的时段。
- 同一观众（多设备、反复进出）重叠时段只计一份时长，只占一个在线名额；
  不同观众即使片段完全相同也分别计数。
- `Stats.Viewers`：窗口内有效观看时长大于 0 的去重观众数。
- `Stats.TotalMillis`：窗口内所有观众有效观看时长总和。
- `Stats.AvgMillis()`：平均观看时长，分母为 `Viewers`，无人观看时为 0。
- `Stats.PeakConcurrent`：窗口内任一时刻在线的不同观众数最大值；
  一人在某时刻离场、另一人恰在该时刻入场，不构成同时在线。

`Tracker` 不是并发安全的，并发调用需上层自行加锁。

## 测试

```sh
go test ./...
```
