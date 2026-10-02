// Package livestats 提供单场直播的观看统计。
// 观看记录与统计结果均保存在进程内存中，由上层直接调用。
package livestats

import "sort"

// Segment 表示一个已配对完成的观看片段，[Start, End)，
// 时间为相对开播的整数毫秒。
type Segment struct {
	Start int64
	End   int64
}

// Stats 是一个时间窗口内的观看统计结果。
type Stats struct {
	// Viewers 窗口内有效观看时长大于 0 的去重观众数。
	Viewers int
	// TotalWatchMs 所有观众在窗口内的有效观看时长总和（毫秒）。
	TotalWatchMs int64
	// AvgWatchMs 平均观看时长，分母为 Viewers，无人观看时为 0。
	AvgWatchMs float64
	// PeakConcurrent 窗口内任一时刻同时在线的不同观众数最大值。
	PeakConcurrent int
}

// Session 表示一场直播的统计会话，保存全部已上报的观看片段。
// 每位观众的片段被归并为互不重叠的有序区间，重复上报天然幂等。
type Session struct {
	viewers map[string][]Segment
}

// New 创建一个空的直播统计会话。
func New() *Session {
	return &Session{viewers: make(map[string][]Segment)}
}

// Report 上报某观众的若干观看片段。
// 片段可乱序、重复或迟到；上报只补充事实，不撤销已有片段。
func (s *Session) Report(viewer string, segs ...Segment) {
	merged := s.viewers[viewer]
	for _, seg := range segs {
		if seg.Start < seg.End {
			merged = append(merged, seg)
		}
	}
	s.viewers[viewer] = union(merged)
}

// Query 查询窗口 [start, end) 内的观看统计。
func (s *Session) Query(start, end int64) Stats {
	var st Stats
	if start >= end {
		return st
	}
	type event struct {
		at    int64
		delta int
	}
	var events []event
	online := 0
	for _, segs := range s.viewers {
		var watch int64
		for _, seg := range segs {
			lo, hi := seg.Start, seg.End
			if lo < start {
				lo = start
			}
			if hi > end {
				hi = end
			}
			if lo >= hi {
				continue
			}
			watch += hi - lo
			events = append(events, event{lo, 1}, event{hi, -1})
		}
		if watch > 0 {
			st.Viewers++
			st.TotalWatchMs += watch
		}
	}
	// 同一时刻先处理离场再处理入场：一人在 t 离场、另一人在 t 入场不构成同时在线。
	sort.Slice(events, func(i, j int) bool {
		if events[i].at != events[j].at {
			return events[i].at < events[j].at
		}
		return events[i].delta < events[j].delta
	})
	for _, e := range events {
		online += e.delta
		if online > st.PeakConcurrent {
			st.PeakConcurrent = online
		}
	}
	if st.Viewers > 0 {
		st.AvgWatchMs = float64(st.TotalWatchMs) / float64(st.Viewers)
	}
	return st
}

// union 将区间切片归并为互不重叠、按起点升序的区间。
func union(segs []Segment) []Segment {
	sort.Slice(segs, func(i, j int) bool {
		if segs[i].Start != segs[j].Start {
			return segs[i].Start < segs[j].Start
		}
		return segs[i].End < segs[j].End
	})
	out := segs[:0]
	for _, seg := range segs {
		if n := len(out); n > 0 && seg.Start <= out[n-1].End {
			if seg.End > out[n-1].End {
				out[n-1].End = seg.End
			}
			continue
		}
		out = append(out, seg)
	}
	return out
}
