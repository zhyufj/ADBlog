package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var adbPath = ".\\adb.exe"

var levelOrder = map[string]int{
	"V": 0,
	"D": 1,
	"I": 2,
	"W": 3,
	"E": 4,
	"F": 5,
}

func main() {
	device := selectDevice()
	if device == "" {
		fmt.Println("未检测到设备")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\n请输入包名 (0=当前APP, 9=列出所有APP, 可留空): ")
	pkg, _ := reader.ReadString('\n')
	pkg = strings.TrimSpace(pkg)

	// 新增功能：输入9列出所有包名
	if pkg == "9" {
		listPackages(device)
		fmt.Print("\n请输入需要抓日志的包名 (可留空): ")
		pkg, _ = reader.ReadString('\n')
		pkg = strings.TrimSpace(pkg)
	}

	// 新增功能：输入0获取当前APP包名
	if pkg == "0" {
		pkg = getCurrentPackage(device)
		if pkg != "" {
			fmt.Println("检测到当前应用:", pkg)
		} else {
			fmt.Println("未检测到当前应用，将显示全部日志")
			pkg = ""
		}
	}

	level := chooseLevel(reader)

	fmt.Println("\n开始抓取日志 (Ctrl+C退出)")
	fmt.Println("设备:", device)
	fmt.Println("最低日志级别:", level)
	if pkg != "" {
		fmt.Println("包名过滤:", pkg)
	}
	fmt.Println("-------------------------------------")

	startLogcat(device, pkg, level)
}

func chooseLevel(reader *bufio.Reader) string {
	fmt.Println("\n选择日志级别:")
	fmt.Println("1. F - 严重")
	fmt.Println("2. E - 错误")
	fmt.Println("3. W - 警告")
	fmt.Println("4. I - 信息")
	fmt.Println("5. D - 调试")
	fmt.Print("选择 (默认2): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		return "F"
	case "3":
		return "W"
	case "4":
		return "I"
	case "5":
		return "D"
	default:
		return "E"
	}
}

func selectDevice() string {
	cmd := exec.Command(adbPath, "devices")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("ADB执行错误:", err)
	}

	lines := strings.Split(string(out), "\n")
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

	if len(devices) == 0 {
		fmt.Println("ADB输出:")
		fmt.Println(string(out))
		return ""
	}

	if len(devices) == 1 {
		fmt.Println("检测到设备:", devices[0])
		return devices[0]
	}

	fmt.Println("\n检测到多个设备:")
	for i, d := range devices {
		fmt.Printf("%d. %s\n", i+1, d)
	}
	fmt.Print("请选择设备: ")

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	var choice int
	fmt.Sscanf(text, "%d", &choice)

	if choice < 1 || choice > len(devices) {
		return devices[0]
	}
	return devices[choice-1]
}

func startLogcat(device, pkg, minLevel string) {
	cmd := exec.Command(adbPath, "-s", device, "logcat")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("logcat启动失败:", err)
		return
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println("logcat启动失败:", err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		processLine(line, pkg, minLevel)
	}
}

func processLine(line, pkg, minLevel string) {
	if pkg != "" && !strings.Contains(line, pkg) {
		return
	}

	level := extractLevel(line)
	if level == "" {
		return
	}

	if levelOrder[level] < levelOrder[minLevel] {
		return
	}

	printColored(line, level)
}

func extractLevel(line string) string {
	if strings.Contains(line, " V ") {
		return "V"
	}
	if strings.Contains(line, " D ") {
		return "D"
	}
	if strings.Contains(line, " I ") {
		return "I"
	}
	if strings.Contains(line, " W ") {
		return "W"
	}
	if strings.Contains(line, " E ") {
		return "E"
	}
	if strings.Contains(line, " F ") {
		return "F"
	}
	return ""
}

func printColored(line, level string) {
	switch level {
	case "F":
		fmt.Println("\033[41;37m" + line + "\033[0m")
	case "E":
		fmt.Println("\033[31m" + line + "\033[0m")
	case "W":
		fmt.Println("\033[33m" + line + "\033[0m")
	case "I":
		fmt.Println("\033[32m" + line + "\033[0m")
	case "D":
		fmt.Println("\033[36m" + line + "\033[0m")
	default:
		fmt.Println(line)
	}
}

// 新增函数：获取当前前台应用的包名
func getCurrentPackage(device string) string {
	cmd := exec.Command(adbPath, "-s", device, "shell", "dumpsys", "window")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "mCurrentFocus") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, "/") {
					return strings.Split(p, "/")[0]
				}
			}
		}
	}
	return ""
}

// 新增函数：列出设备上所有应用的包名
func listPackages(device string) {
	fmt.Println("\n正在获取所有APP包名...\n")
	cmd := exec.Command(adbPath, "-s", device, "shell", "pm", "list", "packages")
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("获取失败:", err)
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			pkg := strings.TrimPrefix(line, "package:")
			fmt.Println(pkg)
		}
	}
}
