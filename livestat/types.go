// Package livestat 提供单场直播的观看统计能力。
// 观看记录与统计结果均保存在进程内存中，由上层直接调用。
package livestat

// Stats 是某一时间窗口内的观看统计结果。
type Stats struct {
	// Viewers 窗口内有效观看时长大于 0 的去重观众数。
	Viewers int
	// TotalMillis 窗口内所有观众的有效观看时长总和（毫秒）。
	// 同一观众的重叠时段只计一份。
	TotalMillis int64
	// PeakConcurrent 窗口内任一时刻同时在线的不同观众数最大值。
	PeakConcurrent int
}

// AvgMillis 返回窗口内观众的平均观看时长（毫秒）。
// 分母为 Stats.Viewers，无人观看时为 0。
func (s Stats) AvgMillis() float64 {
	if s.Viewers == 0 {
		return 0
	}
	return float64(s.TotalMillis) / float64(s.Viewers)
}
