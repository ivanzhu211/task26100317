package livestats

import "testing"

func seg(start, end int64) Segment { return Segment{Start: start, End: end} }

func checkStats(t *testing.T, got Stats, viewers int, total int64, avg float64, peak int) {
	t.Helper()
	if got.Viewers != viewers {
		t.Errorf("Viewers = %d, want %d", got.Viewers, viewers)
	}
	if got.TotalWatchMs != total {
		t.Errorf("TotalWatchMs = %d, want %d", got.TotalWatchMs, total)
	}
	if got.AvgWatchMs != avg {
		t.Errorf("AvgWatchMs = %v, want %v", got.AvgWatchMs, avg)
	}
	if got.PeakConcurrent != peak {
		t.Errorf("PeakConcurrent = %d, want %d", got.PeakConcurrent, peak)
	}
}

// 场景一：甲 [0,10) [5,15)，乙 [15,20)；查询 [0,20)。
func TestBasicWindow(t *testing.T) {
	s := New()
	s.Report("甲", seg(0, 10), seg(5, 15))
	s.Report("乙", seg(15, 20))
	checkStats(t, s.Query(0, 20), 2, 20, 10, 1)
}

// 场景二：迟到补入甲 [8,18)，随后重复补报；再查历史窗口与子窗口。
func TestLateAndDuplicateReport(t *testing.T) {
	s := New()
	s.Report("甲", seg(0, 10), seg(5, 15))
	s.Report("乙", seg(15, 20))

	s.Report("甲", seg(8, 18))
	checkStats(t, s.Query(0, 20), 2, 23, 11.5, 2)

	// 重复补报（含完全重复与被子区间覆盖的片段）不得改变结果。
	s.Report("甲", seg(8, 18))
	s.Report("甲", seg(0, 10), seg(5, 15))
	s.Report("乙", seg(15, 20))
	checkStats(t, s.Query(0, 20), 2, 23, 11.5, 2)

	// 子窗口 [18,20)：甲在 18 已离场，只计入乙。
	checkStats(t, s.Query(18, 20), 1, 2, 2, 1)
}

// 同一时刻一出一进不构成同时在线。
func TestHandoffNotConcurrent(t *testing.T) {
	s := New()
	s.Report("a", seg(0, 10))
	s.Report("b", seg(10, 20))
	checkStats(t, s.Query(0, 20), 2, 20, 10, 1)
}

// 空会话与空窗口。
func TestEmpty(t *testing.T) {
	s := New()
	checkStats(t, s.Query(0, 100), 0, 0, 0, 0)
	s.Report("a", seg(0, 10))
	checkStats(t, s.Query(50, 50), 0, 0, 0, 0)
	checkStats(t, s.Query(20, 10), 0, 0, 0, 0)
}

// 乱序上报与窗口裁剪。
func TestOutOfOrderAndClipping(t *testing.T) {
	s := New()
	s.Report("a", seg(80, 100), seg(0, 20), seg(40, 60))
	checkStats(t, s.Query(0, 100), 1, 60, 60, 1)
	checkStats(t, s.Query(10, 50), 1, 20, 20, 1)
	// 完全落在窗口外的片段不计入。
	checkStats(t, s.Query(100, 200), 0, 0, 0, 0)
}

// 多观众重叠在线的峰值。
func TestPeakOverlap(t *testing.T) {
	s := New()
	s.Report("a", seg(0, 30))
	s.Report("b", seg(10, 20))
	s.Report("c", seg(15, 25))
	checkStats(t, s.Query(0, 30), 3, 50, 50.0/3.0, 3)
}
