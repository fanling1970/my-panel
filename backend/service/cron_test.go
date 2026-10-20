package service

import (
	"testing"
	"time"
)

func TestParseCronValid(t *testing.T) {
	cases := []string{
		"0 8 * * *",     // 每天 8 点
		"30 2 * * 1",    // 每周一 2:30
		"*/15 * * * *",  // 每 15 分钟
		"0 0 1 * *",     // 每月 1 号
		"5 6-8 * * *",   // 每天 6-8 点的第 5 分
		"0 9 * * 1-5",   // 工作日 9 点
		"0,30 7 * * *",  // 每天 7:00 和 7:30
		"@daily",        // 快捷方式
		"@hourly",       // 快捷方式
		"@weekly",       // 快捷方式
		"@monthly",      // 快捷方式
		"7 7 7 7 *",     // 每年 7 月 7 日 7:07
		"0 0 * * 7",     // 7 视为周日
		"60/30 * * * *", // 超界但按范围裁剪不应 panic —— 实际应报错
	}
	for _, c := range cases[:13] {
		if _, err := ParseCron(c); err != nil {
			t.Errorf("ParseCron(%q) 期望成功, 得到 %v", c, err)
		}
	}
	if _, err := ParseCron("60/30 * * * *"); err == nil {
		t.Errorf("ParseCron(60/30 * * * *) 期望失败(分钟超界)")
	}
}

func TestParseCronInvalid(t *testing.T) {
	cases := []string{
		"",
		"* * * *",       // 字段不足
		"* * * * * *",   // 字段过多
		"a * * * *",     // 非数字
		"*/0 * * * *",   // 步长为 0
		"0-100 * * * *", // 范围超界
		"@year",         // 未知快捷方式（按字段解析会失败）
	}
	for _, c := range cases {
		if _, err := ParseCron(c); err == nil {
			t.Errorf("ParseCron(%q) 期望失败, 却成功了", c)
		}
	}
}

func TestNextDaily(t *testing.T) {
	expr, err := ParseCron("30 8 * * *")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 10, 20, 9, 0, 0, 0, time.Local)
	next := expr.Next(from)
	want := time.Date(2026, 10, 21, 8, 30, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Errorf("Next = %v, want %v", next, want)
	}

	// 当天还没到 8:30
	from2 := time.Date(2026, 10, 20, 8, 0, 0, 0, time.Local)
	next2 := expr.Next(from2)
	want2 := time.Date(2026, 10, 20, 8, 30, 0, 0, time.Local)
	if !next2.Equal(want2) {
		t.Errorf("Next = %v, want %v", next2, want2)
	}
}

func TestNextWeekly(t *testing.T) {
	expr, err := ParseCron("0 9 * * 1") // 每周一 9 点
	if err != nil {
		t.Fatal(err)
	}
	// 2026-10-20 是周二
	from := time.Date(2026, 10, 20, 12, 0, 0, 0, time.Local)
	next := expr.Next(from)
	want := time.Date(2026, 10, 26, 9, 0, 0, 0, time.Local) // 下周一
	if !next.Equal(want) {
		t.Errorf("Next = %v, want %v", next, want)
	}
}

func TestNextEvery15Minutes(t *testing.T) {
	expr, err := ParseCron("*/15 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 10, 20, 10, 7, 30, 0, time.Local)
	next := expr.Next(from)
	want := time.Date(2026, 10, 20, 10, 15, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Errorf("Next = %v, want %v", next, want)
	}
}

func TestDayMatchDomDowOR(t *testing.T) {
	// 标准cron: 日和周都受限时, 命中其一即执行
	expr, err := ParseCron("0 0 1 * 1") // 1 号或周一
	if err != nil {
		t.Fatal(err)
	}
	// 2026-10-19 是周一（非 1 号）→ 应命中
	if !expr.dayMatch(time.Date(2026, 10, 19, 0, 0, 0, 0, time.Local)) {
		t.Error("周一应命中 (1号 或 周一)")
	}
	// 2026-10-2 是周五非 1 号 → 不命中
	if expr.dayMatch(time.Date(2026, 10, 2, 0, 0, 0, 0, time.Local)) {
		t.Error("普通周五不应命中")
	}
}
