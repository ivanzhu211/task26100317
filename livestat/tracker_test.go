package livestat

import "testing"

func checkStats(t *testing.T, got Stats, viewers int, total int64, avg float64, peak int) {
	t.Helper()
	if got.Viewers != viewers {
		t.Errorf("Viewers = %d, want %d", got.Viewers, viewers)
	}
	if got.TotalMillis != total {
		t.Errorf("TotalMillis = %d, want %d", got.TotalMillis, total)
	}
	if got.AvgMillis() != avg {
		t.Errorf("AvgMillis() = %v, want %v", got.AvgMillis(), avg)
	}
	if got.PeakConcurrent != peak {
		t.Errorf("PeakConcurrent = %d, want %d", got.PeakConcurrent, peak)
	}
}

// 需求场景：甲 [0,10) [5,15)，乙 [15,20)；查询 [0,20)。
// 随后迟到补入甲 [8,18)，重复补报结果不变；再查 [18,20) 只计入乙。
func TestRequiredScenario(t *testing.T) {
	tr := NewTracker()
	tr.Report("甲", 0, 10)
	tr.Report("甲", 5, 15)
	tr.Report("乙", 15, 20)

	// 甲去重后 [0,15) 计 15ms，乙 [15,20) 计 5ms。
	// 15ms 时刻甲离场、乙入场，不构成同时在线，峰值为 1。
	checkStats(t, tr.Query(0, 20), 2, 20, 10, 1)

	// 迟到补入甲 [8,18)：甲合并为 [0,18) 计 18ms，总计 23ms，峰值 2。
	tr.Report("甲", 8, 18)
	checkStats(t, tr.Query(0, 20), 2, 23, 11.5, 2)

	// 重复补报不改变任何统计结果。
	tr.Report("甲", 8, 18)
	tr.Report("甲", 0, 10)
	checkStats(t, tr.Query(0, 20), 2, 23, 11.5, 2)

	// [18,20) 只计入乙：1 人、2ms、峰值 1。
	checkStats(t, tr.Query(18, 20), 1, 2, 2, 1)
}

func TestEmptyAndZeroWindows(t *testing.T) {
	tr := NewTracker()
	checkStats(t, tr.Query(0, 100), 0, 0, 0, 0)

	tr.Report("甲", 10, 20)
	// 空窗口与倒置窗口。
	checkStats(t, tr.Query(10, 10), 0, 0, 0, 0)
	checkStats(t, tr.Query(20, 10), 0, 0, 0, 0)
	// 窗口与片段不相交。
	checkStats(t, tr.Query(0, 10), 0, 0, 0, 0)
	checkStats(t, tr.Query(20, 30), 0, 0, 0, 0)
	// 无效片段被忽略。
	tr.Report("乙", 30, 30)
	tr.Report("乙", 40, 35)
	checkStats(t, tr.Query(0, 100), 1, 10, 10, 1)
}

func TestOutOfOrderAndDuplicateReports(t *testing.T) {
	tr := NewTracker()
	tr.Report("甲", 50, 60)
	tr.Report("甲", 10, 20)
	tr.Report("甲", 30, 40)
	tr.Report("甲", 10, 20) // 完全重复
	tr.Report("甲", 12, 18) // 被已有区间包含
	checkStats(t, tr.Query(0, 100), 1, 30, 30, 1)

	// 桥接片段把三段合并为 [10,60)。
	tr.Report("甲", 15, 55)
	checkStats(t, tr.Query(0, 100), 1, 50, 50, 1)
}

func TestPeakConcurrentSameInstantHandoff(t *testing.T) {
	tr := NewTracker()
	tr.Report("甲", 0, 10)
	tr.Report("乙", 10, 20)
	// 甲在 10 离场、乙在 10 入场，峰值仍为 1。
	checkStats(t, tr.Query(0, 20), 2, 20, 10, 1)

	tr.Report("丙", 5, 15)
	checkStats(t, tr.Query(0, 20), 3, 30, 10, 2)
}

func TestWindowClipping(t *testing.T) {
	tr := NewTracker()
	tr.Report("甲", 0, 100)
	tr.Report("乙", 40, 60)
	// 窗口裁剪片段两端。
	checkStats(t, tr.Query(30, 70), 2, 60, 30, 2)
	// 峰值只统计窗口内的重叠。
	checkStats(t, tr.Query(0, 40), 1, 40, 40, 1)
	checkStats(t, tr.Query(60, 100), 1, 40, 40, 1)
}

func TestSameViewerMultipleDevicesOverlap(t *testing.T) {
	tr := NewTracker()
	// 同一观众多设备同时在线，只计一份时长、一个在线名额。
	tr.Report("甲", 0, 10)
	tr.Report("甲", 0, 10)
	tr.Report("乙", 5, 15)
	checkStats(t, tr.Query(0, 20), 2, 20, 10, 2)
}
