package node

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"log"
	"time"
)

type CpuStats struct {
	Count  int
	Usages []float64
	Info   []cpu.InfoStat
}

type DiskStats struct {
	Partitions []disk.PartitionStat
}

type MemoryStats struct {
	Total        uint64
	Usage        uint64
	UsagePercent float64
	Available    uint64
}

type Stats struct {
	Cpu    CpuStats
	Memory MemoryStats
	Disk   DiskStats
}

func GetStats() *Stats {
	cpuStats := getCpuInfo()
	diskStats := getDiskInfo()
	memoryStats := getMemoryInfo()

	return &Stats{
		Cpu:    cpuStats,
		Disk:   diskStats,
		Memory: memoryStats,
	}
}

func getCpuInfo() CpuStats {
	percent, err := cpu.Percent(time.Second, true)
	if err != nil {
		log.Printf("Error getting CPU info: %v\n", err)
	}

	cpuCnt, err := cpu.Counts(true)
	if err != nil {
		log.Printf("Error getting CPU count: %v\n", err)
	}

	info, err := cpu.Info()
	if err != nil {
		log.Printf("Error getting CPU info: %v\n", err)
	}

	return CpuStats{
		Count:  cpuCnt,
		Usages: percent,
		Info:   info,
	}
}

func getDiskInfo() DiskStats {
	partitions, err := disk.Partitions(true)
	if err != nil {
		log.Printf("Error getting disk info: %v\n", err)
	}

	return DiskStats{
		Partitions: partitions,
	}
}

func getMemoryInfo() MemoryStats {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory info: %v\n", err)
	}

	return MemoryStats{
		Total:        vmStat.Total,
		Available:    vmStat.Available,
		Usage:        vmStat.Used,
		UsagePercent: vmStat.UsedPercent,
	}
}
