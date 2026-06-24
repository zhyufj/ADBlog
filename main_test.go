package main

import (
	"bufio"
	"strings"
	"testing"
)

// ==================== extractLevel 单元测试 ====================

func TestExtractLevel_BriefFormat(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		level string
	}{
		{"V级别-brief", "V/System: verbose message", "V"},
		{"D级别-brief", "D/AudioFlinger: debug message", "D"},
		{"I级别-brief", "I/ActivityManager: info message", "I"},
		{"W级别-brief", "W/PackageManager: warning message", "W"},
		{"E级别-brief", "E/AndroidRuntime: error message", "E"},
		{"F级别-brief", "F/libc: fatal signal 11", "F"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractLevel(tt.line); got != tt.level {
				t.Errorf("extractLevel(%q) = %q, want %q", tt.line, got, tt.level)
			}
		})
	}
}

func TestExtractLevel_ThreadtimeFormat(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		level string
	}{
		{"V级别-threadtime", "06-22 10:30:45.123   456   789 V System: verbose", "V"},
		{"D级别-threadtime", "06-22 10:30:45.123   456   789 D AudioFlinger: debug", "D"},
		{"I级别-threadtime", "06-22 10:30:45.123   456   789 I ActivityManager: info", "I"},
		{"W级别-threadtime", "06-22 10:30:45.123   456   789 W PackageManager: warn", "W"},
		{"E级别-threadtime", "06-22 10:30:45.123   456   789 E AndroidRuntime: err", "E"},
		{"F级别-threadtime", "06-22 10:30:45.123   456   789 F libc: fatal", "F"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractLevel(tt.line); got != tt.level {
				t.Errorf("extractLevel(%q) = %q, want %q", tt.line, got, tt.level)
			}
		})
	}
}

// TestExtractLevel_MessageBodyFalseMatch 验证消息体中的级别标记不会被误判
// 这是修复的核心 Bug：E 级别日志消息中含 "I/Stream" 不应被误判为 I
func TestExtractLevel_MessageBodyFalseMatch(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		level string
	}{
		{
			"E级别消息含I/不应误判为I",
			"E/MyApp(12345): Error reading I/Stream data",
			"E",
		},
		{
			"E级别消息含D/不应误判为D",
			"E/MyApp(12345): Failed D/Config load",
			"E",
		},
		{
			"W级别消息含E/不应误判为E",
			"W/MyApp(12345): caused by E/Network error",
			"W",
		},
		{
			"I级别消息含W/不应误判为W",
			"I/MyApp(12345): processed W/Write complete",
			"I",
		},
		{
			"threadtime-E级别消息含I/不应误判为I",
			"06-22 10:30:45.123   456   789 E MyApp: Error reading I/Stream",
			"E",
		},
		{
			"threadtime-W级别消息含E/不应误判为E",
			"06-22 10:30:45.123   456   789 W MyApp: caused by E/Network failure",
			"W",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractLevel(tt.line); got != tt.level {
				t.Errorf("extractLevel(%q) = %q, want %q (误判！消息体中的级别标记被错误识别)", tt.line, got, tt.level)
			}
		})
	}
}

func TestExtractLevel_NoLevelMarker(t *testing.T) {
	tests := []string{
		"",
		"just a plain message",
		"   at com.example.MyClass.method(MyClass.java:123)",
		"1234567890",
		"--------- beginning of main",
		"--------- beginning of system",
	}
	for _, line := range tests {
		if got := extractLevel(line); got != "" {
			t.Errorf("extractLevel(%q) = %q, want \"\" (无级别标记应返回空)", line, got)
		}
	}
}

func TestExtractLevel_LevelBeyond50Chars(t *testing.T) {
	// 构造一条日志，级别标记在第51个字符之后
	// 前50个字符无级别标记
	prefix := strings.Repeat("x", 51)
	line := prefix + " E/MyApp: error"
	if got := extractLevel(line); got != "" {
		t.Errorf("extractLevel(long line with level beyond 50) = %q, want \"\"", got)
	}
}

func TestExtractLevel_LevelAt50Boundary(t *testing.T) {
	// 级别标记恰好在第49-50字符位置（50字符范围内）
	prefix := strings.Repeat("x", 48) // 占48个字符
	line := prefix + " E/MyApp: error" // " E" 在第49-50位置
	_ = line
	// 构造更精确的：让 " E/" 的前50字符中包含 " E/"
	prefix2 := strings.Repeat("x", 47)
	line2 := prefix2 + " E/MyApp: error" // " E/" 从48开始，" E" 在48-49, "/" 在49
	if got := extractLevel(line2); got != "E" {
		t.Errorf("extractLevel(level at boundary) = %q, want \"E\"", got)
	}
}

func TestExtractLevel_LevelCharInTagName(t *testing.T) {
	// Tag名或消息体中包含级别字符，但不构成「空格+字母+/或空格」的结构，不应被匹配
	tests := []string{
		"FIREBASE: connection failed",     // F在单词中，无前导空格
		"myapp processE error occurred",   // E在单词中，无前导空格
		"value=D1 result",                 // D前为=而非空格
	}
	for _, line := range tests {
		if got := extractLevel(line); got != "" {
			t.Errorf("extractLevel(%q) = %q, want \"\" (非级别位置的字母不应匹配)", line, got)
		}
	}
}

// ==================== levelOrder 映射表测试 ====================

func TestLevelOrder(t *testing.T) {
	if levelOrder["V"] != 0 {
		t.Error("V should be 0")
	}
	if levelOrder["D"] != 1 {
		t.Error("D should be 1")
	}
	if levelOrder["I"] != 2 {
		t.Error("I should be 2")
	}
	if levelOrder["W"] != 3 {
		t.Error("W should be 3")
	}
	if levelOrder["E"] != 4 {
		t.Error("E should be 4")
	}
	if levelOrder["F"] != 5 {
		t.Error("F should be 5")
	}
}

// ==================== chooseLevel 测试 ====================

func TestChooseLevel_AllValidInputs(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1\n", "F"},
		{"2\n", "E"},
		{"3\n", "W"},
		{"4\n", "I"},
		{"5\n", "D"},
		{"6\n", "V"},
	}
	for _, tt := range tests {
		t.Run("input_"+tt.input, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			if got := chooseLevel(reader); got != tt.expected {
				t.Errorf("chooseLevel(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestChooseLevel_DefaultCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"空输入(回车默认)", "\n"},
		{"无效数字7", "7\n"},
		{"无效数字0", "0\n"},
		{"负数", "-1\n"},
		{"字母", "abc\n"},
		{"特殊字符", "!@#\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			if got := chooseLevel(reader); got != "E" {
				t.Errorf("chooseLevel(%q) = %q, want \"E\" (默认值)", tt.input, got)
			}
		})
	}
}

// ==================== processLine 过滤逻辑测试 ====================

func TestProcessLine_FilterByPackage(t *testing.T) {
	// processLine 内部调用 printColored 输出到 stdout，无法直接捕获。
	// 这里测试 extractLevel + levelOrder 的组合逻辑（processLine 的核心判断）。
	// 模拟 processLine 中的三条件判断：
	// 1. pkg != "" && !strings.Contains(line, pkg) → return
	// 2. level == "" → return
	// 3. levelOrder[level] < levelOrder[minLevel] → return

	tests := []struct {
		name     string
		line     string
		pkg      string
		minLevel string
		shouldShow bool // 是否应该被显示
	}{
		// 包名匹配 + 级别足够
		{"包名匹配+级别足够", "I/com.example.app: info", "com.example.app", "I", true},
		{"包名匹配+级别高于阈值", "E/com.example.app: error", "com.example.app", "I", true},
		// 包名不匹配
		{"包名不匹配", "I/com.other.app: info", "com.example.app", "I", false},
		// 级别低于阈值
		{"级别低于阈值", "D/com.example.app: debug", "com.example.app", "I", false},
		// 无级别标记
		{"无级别标记", "plain text without marker", "", "V", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 processLine 的判断逻辑
			if tt.pkg != "" && !strings.Contains(tt.line, tt.pkg) {
				if tt.shouldShow {
					t.Error("包名不匹配，应被过滤")
				}
				return
			}
			level := extractLevel(tt.line)
			if level == "" {
				if tt.shouldShow {
					t.Error("无级别标记，应被过滤")
				}
				return
			}
			if levelOrder[level] < levelOrder[tt.minLevel] {
				if tt.shouldShow {
					t.Error("级别低于阈值，应被过滤")
				}
				return
			}
			if !tt.shouldShow {
				t.Error("日志应被显示但被过滤了")
			}
		})
	}
}

// ==================== selectDevice 设备解析逻辑测试 ====================

func TestSelectDevice_ParseADBOutput(t *testing.T) {
	// 模拟 selectDevice 中对 adb devices 输出的解析逻辑
	adbOutput := "List of devices attached\nABC123\tdevice\nDEF456\tdevice\nGHI789\toffline\nJKL012\tunauthorized\n\n"
	lines := strings.Split(adbOutput, "\n")
	var devices []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" {
			devices = append(devices, fields[0])
		}
	}

	if len(devices) != 2 {
		t.Errorf("应识别2台设备，实际: %d", len(devices))
	}
	if devices[0] != "ABC123" || devices[1] != "DEF456" {
		t.Errorf("设备列表不正确: %v", devices)
	}
}

// ==================== getCurrentPackage 解析测试 ====================

func TestGetCurrentPackage_ParseDumpsysOutput(t *testing.T) {
	// 模拟 mCurrentFocus 解析逻辑
	dumpsysOutput := `mCurrentFocus=Window{abc123 u0 com.example.app/com.example.MainActivity}
mFocusedApp=...`

	lines := strings.Split(dumpsysOutput, "\n")
	var pkg string
	for _, line := range lines {
		if strings.Contains(line, "mCurrentFocus") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, "/") {
					pkg = strings.Split(p, "/")[0]
				}
			}
		}
	}

	if pkg != "com.example.app" {
		t.Errorf("应识别包名 com.example.app，实际: %q", pkg)
	}
}

func TestGetCurrentPackage_NoFocus(t *testing.T) {
	dumpsysOutput := `mFocusedApp=something
other info`

	lines := strings.Split(dumpsysOutput, "\n")
	var pkg string
	for _, line := range lines {
		if strings.Contains(line, "mCurrentFocus") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, "/") {
					pkg = strings.Split(p, "/")[0]
				}
			}
		}
	}

	if pkg != "" {
		t.Errorf("无 mCurrentFocus 时应返回空，实际: %q", pkg)
	}
}

// ==================== listPackages 解析测试 ====================

func TestListPackages_ParsePMOutput(t *testing.T) {
	pmOutput := "package:com.android.settings\npackage:com.tencent.mm\npackage:com.example.app\n"
	lines := strings.Split(pmOutput, "\n")
	var packages []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			pkg := strings.TrimPrefix(line, "package:")
			packages = append(packages, pkg)
		}
	}

	if len(packages) != 3 {
		t.Errorf("应解析出3个包名，实际: %d", len(packages))
	}
	expected := []string{"com.android.settings", "com.tencent.mm", "com.example.app"}
	for i, pkg := range packages {
		if pkg != expected[i] {
			t.Errorf("packages[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}
