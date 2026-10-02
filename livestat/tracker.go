package livestat

import "sort"

// interval 表示一个半开区间 [start, end)。
type interval struct {
	start, end int64
}

// Tracker 记录单场直播的观看片段并支持任意窗口的统计查询。
// 不是并发安全的；如需并发调用，由上层自行加锁。
type Tracker struct {
	// viewers 按观众标识保存已合并、互不重叠且有序的观看区间。
	viewers map[string][]interval
}

// NewTracker 创建一个空的直播观看统计器。
func NewTracker() *Tracker {
	return &Tracker{viewers: make(map[string][]interval)}
}

// Report 上报一个已完成配对的观看片段 [startMillis, endMillis)。
// 片段可以乱序、重复或迟到；上报只补充事实，重复上报不改变统计结果。
func (t *Tracker) Report(viewerID string, startMillis, endMillis int64) {
	if endMillis <= startMillis {
		return
	}
	ivs := t.viewers[viewerID]
	// 找到与 [startMillis, endMillis] 相邻或重叠的区间范围，合并为一个区间。
	lo := sort.Search(len(ivs), func(i int) bool { return ivs[i].end >= startMillis })
	hi := sort.Search(len(ivs), func(i int) bool { return ivs[i].start > endMillis })
	merged := interval{start: startMillis, end: endMillis}
	if lo < len(ivs) && ivs[lo].start < merged.start {
		merged.start = ivs[lo].start
	}
	if hi > 0 && ivs[hi-1].end > merged.end {
		merged.end = ivs[hi-1].end
	}
	mergedSet := make([]interval, 0, len(ivs)-(hi-lo)+1)
	mergedSet = append(mergedSet, ivs[:lo]...)
	mergedSet = append(mergedSet, merged)
	mergedSet = append(mergedSet, ivs[hi:]...)
	t.viewers[viewerID] = mergedSet
}

// Query 查询窗口 [startMillis, endMillis) 内的观看统计。
// 只统计已上报的时段；同一观众重叠时段只计一份，也只占一个在线名额。
func (t *Tracker) Query(startMillis, endMillis int64) Stats {
	var stats Stats
	if endMillis <= startMillis {
		return stats
	}
	// 扫描线事件：+1 入场，-1 离场。
	type event struct {
		at    int64
		delta int
	}
	var events []event
	for _, ivs := range t.viewers {
		var watchMillis int64
		for _, iv := range ivs {
			s := max(iv.start, startMillis)
			e := min(iv.end, endMillis)
			if e <= s {
				continue
			}
			watchMillis += e - s
			events = append(events, event{at: s, delta: 1}, event{at: e, delta: -1})
		}
		if watchMillis > 0 {
			stats.Viewers++
			stats.TotalMillis += watchMillis
		}
	}
	// 同一时刻先处理离场再处理入场：一人在 t 离场、另一人在 t 入场不构成同时在线。
	sort.Slice(events, func(i, j int) bool {
		if events[i].at != events[j].at {
			return events[i].at < events[j].at
		}
		return events[i].delta < events[j].delta
	})
	online := 0
	for _, ev := range events {
		online += ev.delta
		if ev.delta > 0 && online > stats.PeakConcurrent {
			stats.PeakConcurrent = online
		}
	}
	return stats
}
