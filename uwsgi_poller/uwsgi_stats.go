package uwsgi_poller

import "fmt"

type UwsgiStats struct {
	Cwd               string `json:"cwd"`
	Gid               int    `json:"gid"`
	ListenQueue       int    `json:"listen_queue"`
	ListenQueueErrors int    `json:"listen_queue_errors"`
	Load              int    `json:"load"`
	Locks             []struct {
		User_0 int `json:"user 0"`
	} `json:"locks"`
	Pid         int `json:"pid"`
	SignalQueue int `json:"signal_queue"`
	Sockets     []struct {
		CanOffload int    `json:"can_offload"`
		Name       string `json:"name"`
		Proto      string `json:"proto"`
		Queue      int    `json:"queue"`
		Shared     int    `json:"shared"`
	} `json:"sockets"`
	UID     int    `json:"uid"`
	Version string `json:"version"`
	Workers []struct {
		Apps []struct {
			Chdir       string `json:"chdir"`
			Exceptions  int    `json:"exceptions"`
			ID          int    `json:"id"`
			Modifier1   int    `json:"modifier1"`
			Mountpoint  string `json:"mountpoint"`
			Requests    int    `json:"requests"`
			StartupTime int    `json:"startup_time"`
		} `json:"apps"`
		AvgRt int `json:"avg_rt"`
		Cores []struct {
			ID                int           `json:"id"`
			InRequest         int           `json:"in_request"`
			OffloadedRequests int           `json:"offloaded_requests"`
			Requests          int           `json:"requests"`
			RoutedRequests    int           `json:"routed_requests"`
			StaticRequests    int           `json:"static_requests"`
			Vars              []interface{} `json:"vars"`
			WriteErrors       int           `json:"write_errors"`
		} `json:"cores"`
		DeltaRequests int    `json:"delta_requests"`
		Exceptions    int    `json:"exceptions"`
		HarakiriCount int    `json:"harakiri_count"`
		ID            int    `json:"id"`
		LastSpawn     int    `json:"last_spawn"`
		Pid           int    `json:"pid"`
		Requests      int    `json:"requests"`
		RespawnCount  int    `json:"respawn_count"`
		Rss           int    `json:"rss"`
		RunningTime   int    `json:"running_time"`
		SignalQueue   int    `json:"signal_queue"`
		Signals       int    `json:"signals"`
		Status        string `json:"status"`
		Tx            int    `json:"tx"`
		Vsz           int    `json:"vsz"`
	} `json:"workers"`
}

func (s *UwsgiStats) UniqueID() string {
	return fmt.Sprintf("%s-%d-%d-%d",
		s.Cwd,
		s.UID,
		s.Pid,
		s.Gid)
}

func (s *UwsgiStats) TotalWorkers() float64 {
	return float64(len(s.Workers))
}

func (s *UwsgiStats) ExceptionsCount() (n float64) {
	for _, wk := range s.Workers {
		n += float64(wk.Exceptions)
	}
	return n
}

func (s *UwsgiStats) BusyWorkers() (n float64) {
	for _, wk := range s.Workers {
		if wk.Status == "busy" {
			n += 1.0
		}
	}
	return n
}

func (s *UwsgiStats) IdleWorkers() (n float64) {
	for _, wk := range s.Workers {
		if wk.Status == "idle" {
			n += 1.0
		}
	}
	return n
}

func (s *UwsgiStats) BusyWorkersPercentage() (n float64) {
	total_workers := s.TotalWorkers()
	if total_workers == 0 {
		return 0.0
	}
	busy_workers := s.BusyWorkers()
	return (busy_workers * 100.0) / total_workers
}

func (s *UwsgiStats) IdleWorkersPercentage() (n float64) {
	total_workers := s.TotalWorkers()
	if total_workers == 0 {
		return 0.0
	}
	idle_workers := s.IdleWorkers()
	return (idle_workers * 100.0) / total_workers
}

func (s *UwsgiStats) String() string {
	return fmt.Sprintf(
		"Load %d Pid %d Workers %d",
		s.Load,
		s.Pid,
		len(s.Workers),
	)
}
