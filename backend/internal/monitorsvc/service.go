package monitorsvc

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"zshell/backend/internal/model"
	"zshell/backend/internal/sshsvc"
)

type Service struct {
	mu      sync.Mutex
	samples map[string]netSample
}

type Snapshot struct {
	UpdatedAt  string       `json:"updatedAt"`
	Host       HostInfo     `json:"host"`
	Loads      LoadInfo     `json:"loads"`
	System     SystemInfo   `json:"system"`
	Processes  []ProcessRow `json:"processes"`
	Networks   []NetworkRow `json:"networks"`
	Partitions []Partition  `json:"partitions"`
}

type HostInfo struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

type LoadInfo struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	DiskPercent   float64 `json:"diskPercent"`
	Load1         float64 `json:"load1"`
	Cores         int     `json:"cores"`
	MemoryUsedMB  int64   `json:"memoryUsedMB"`
	MemoryTotalMB int64   `json:"memoryTotalMB"`
}

type SystemInfo struct {
	ServerTime    string `json:"serverTime"`
	TimeZone      string `json:"timeZone"`
	OS            string `json:"os"`
	Kernel        string `json:"kernel"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
}

type ProcessRow struct {
	MemoryMB   float64 `json:"memoryMB"`
	CPUPercent float64 `json:"cpuPercent"`
	Name       string  `json:"name"`
}

type NetworkRow struct {
	Name    string  `json:"name"`
	RxBps   float64 `json:"rxBps"`
	TxBps   float64 `json:"txBps"`
	RxTotal int64   `json:"rxTotal"`
	TxTotal int64   `json:"txTotal"`
}

type Partition struct {
	FileSystem string  `json:"fileSystem"`
	Mount      string  `json:"mount"`
	TotalBytes int64   `json:"totalBytes"`
	FreeBytes  int64   `json:"freeBytes"`
	UsePercent float64 `json:"usePercent"`
}

type rawNet struct {
	name string
	rx   int64
	tx   int64
}

type netSample struct {
	at time.Time
	rx int64
	tx int64
}

func NewService() *Service {
	return &Service{samples: make(map[string]netSample)}
}

func (s *Service) Snapshot(conn model.Connection, processSort string, timeout time.Duration) (Snapshot, error) {
	sortFlag := "-rss"
	if strings.EqualFold(processSort, "cpu") {
		sortFlag = "-pcpu"
	}

	cmd := monitorCommand(sortFlag)
	result, err := sshsvc.ExecCommandShared(conn, cmd, timeout)
	if err != nil {
		return Snapshot{}, err
	}
	if result.ExitCode != 0 {
		return Snapshot{}, fmt.Errorf("monitor command exit %d: %s", result.ExitCode, strings.TrimSpace(result.Stderr))
	}

	snapshot, nets, err := parseMonitorOutput(result.Stdout)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	snapshot.Host.Address = conn.Host
	snapshot.Host.Port = conn.Port
	snapshot.Host.Username = conn.Username
	snapshot.Networks = s.networkRates(conn.ID, nets)

	return snapshot, nil
}

func monitorCommand(sortFlag string) string {
	return fmt.Sprintf(`bash -lc 'set +e
hostname | awk "{print \"HOST\t\" \$0}"
cores=$(getconf _NPROCESSORS_ONLN 2>/dev/null); [ -z "$cores" ] && cores=1
read l1 l5 l15 rest < /proc/loadavg
printf "LOAD\t%%s\t%%s\n" "$cores" "$l1"
awk "/MemTotal:/ {t=\$2} /MemAvailable:/ {a=\$2} END {u=t-a; p=(t>0)?u*100/t:0; printf \"MEM\t%%d\t%%d\t%%.2f\n\", t, a, p}" /proc/meminfo
df -P -B1 / 2>/dev/null | awk "NR==2 {gsub(/%%/,\"\",\$5); printf \"DISK\t%%s\n\", \$5}"
server_time=$(date -Is 2>/dev/null)
timezone=$(date +%%Z 2>/dev/null)
uptime_seconds=$(awk "{printf \"%%.0f\", \$1}" /proc/uptime 2>/dev/null)
os_pretty=$(awk -F= "/^PRETTY_NAME=/{gsub(/\"/,\"\",\$2); print \$2; exit}" /etc/os-release 2>/dev/null)
kernel=$(uname -srmo 2>/dev/null)
printf "TIME\t%%s\t%%s\n" "$server_time" "$timezone"
printf "SYS\t%%s\t%%s\t%%s\n" "$os_pretty" "$kernel" "$uptime_seconds"
ps -eo rss=,pcpu=,comm= --sort=%s 2>/dev/null | head -n 5 | awk "{printf \"PROC\t%%s\t%%s\t\", \$1, \$2; for(i=3;i<=NF;i++){printf (i==3?\"%%s\":\" %%s\"), \$i}; printf \"\n\"}"
awk -F"[: ]+" "NR>2 && \$2 != \"lo\" {printf \"NET\t%%s\t%%s\t%%s\n\", \$2, \$3, \$11}" /proc/net/dev
df -P -B1 2>/dev/null | awk "NR>1 {gsub(/%%/,\"\",\$5); printf \"FS\t%%s\t%%s\t%%s\t%%s\t%%s\n\", \$1, \$2, \$4, \$5, \$6}"'`, sortFlag)
}

func parseMonitorOutput(output string) (Snapshot, []rawNet, error) {
	var snapshot Snapshot
	var nets []rawNet

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		switch fields[0] {
		case "HOST":
			if len(fields) > 1 {
				snapshot.Host.Name = fields[1]
			}
		case "LOAD":
			if len(fields) >= 3 {
				cores := parseInt(fields[1])
				load1 := parseFloat(fields[2])
				snapshot.Loads.Cores = int(cores)
				snapshot.Loads.Load1 = load1
				if cores > 0 {
					snapshot.Loads.CPUPercent = clampPercent(load1 * 100 / float64(cores))
				}
			}
		case "MEM":
			if len(fields) >= 4 {
				totalKB := parseInt(fields[1])
				availableKB := parseInt(fields[2])
				snapshot.Loads.MemoryPercent = clampPercent(parseFloat(fields[3]))
				snapshot.Loads.MemoryTotalMB = totalKB / 1024
				snapshot.Loads.MemoryUsedMB = (totalKB - availableKB) / 1024
			}
		case "DISK":
			if len(fields) >= 2 {
				snapshot.Loads.DiskPercent = clampPercent(parseFloat(fields[1]))
			}
		case "TIME":
			if len(fields) >= 3 {
				snapshot.System.ServerTime = fields[1]
				snapshot.System.TimeZone = fields[2]
			}
		case "SYS":
			if len(fields) >= 4 {
				snapshot.System.OS = fields[1]
				snapshot.System.Kernel = fields[2]
				snapshot.System.UptimeSeconds = parseInt(fields[3])
			}
		case "PROC":
			if len(fields) >= 4 {
				snapshot.Processes = append(snapshot.Processes, ProcessRow{
					MemoryMB:   parseFloat(fields[1]) / 1024,
					CPUPercent: parseFloat(fields[2]),
					Name:       fields[3],
				})
			}
		case "NET":
			if len(fields) >= 4 {
				nets = append(nets, rawNet{
					name: fields[1],
					rx:   parseInt(fields[2]),
					tx:   parseInt(fields[3]),
				})
			}
		case "FS":
			if len(fields) >= 6 {
				snapshot.Partitions = append(snapshot.Partitions, Partition{
					FileSystem: fields[1],
					TotalBytes: parseInt(fields[2]),
					FreeBytes:  parseInt(fields[3]),
					UsePercent: clampPercent(parseFloat(fields[4])),
					Mount:      fields[5],
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Snapshot{}, nil, err
	}

	return snapshot, nets, nil
}

func (s *Service) networkRates(connectionID string, nets []rawNet) []NetworkRow {
	now := time.Now()
	rows := make([]NetworkRow, 0, len(nets)+1)
	var totalRx int64
	var totalTx int64

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, netItem := range nets {
		totalRx += netItem.rx
		totalTx += netItem.tx

		key := connectionID + ":" + netItem.name
		previous, ok := s.samples[key]
		s.samples[key] = netSample{at: now, rx: netItem.rx, tx: netItem.tx}

		row := NetworkRow{Name: netItem.name, RxTotal: netItem.rx, TxTotal: netItem.tx}
		if ok {
			seconds := now.Sub(previous.at).Seconds()
			if seconds > 0 {
				row.RxBps = positiveRate(netItem.rx-previous.rx, seconds)
				row.TxBps = positiveRate(netItem.tx-previous.tx, seconds)
			}
		}
		rows = append(rows, row)
	}

	totalKey := connectionID + ":__total"
	previous, ok := s.samples[totalKey]
	s.samples[totalKey] = netSample{at: now, rx: totalRx, tx: totalTx}
	total := NetworkRow{Name: "total", RxTotal: totalRx, TxTotal: totalTx}
	if ok {
		seconds := now.Sub(previous.at).Seconds()
		if seconds > 0 {
			total.RxBps = positiveRate(totalRx-previous.rx, seconds)
			total.TxBps = positiveRate(totalTx-previous.tx, seconds)
		}
	}

	return append([]NetworkRow{total}, rows...)
}

func parseInt(value string) int64 {
	parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return parsed
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return parsed
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func positiveRate(delta int64, seconds float64) float64 {
	if delta <= 0 {
		return 0
	}
	return float64(delta) / seconds
}
