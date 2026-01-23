package domain

import "time"

type PodCrash struct {
	Namespace     string
	PodName       string
	ContainerName string
	ExitCode      int32
	Reason        string
	Signal        int32
	RestartCount  int32
	StartedAt     time.Time
	FinishedAt    time.Time
}

func NewPodCrash(namespace, podName, containerName string) *PodCrash {
	return &PodCrash{
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
	}
}

func (p *PodCrash) IsOOMKilled() bool {
	return p.Reason == "OOMKilled"
}

func (p *PodCrash) IsCrashLoopBackOff() bool {
	return p.Reason == "CrashLoopBackOff"
}

func (p *PodCrash) FullName() string {
	return p.Namespace + "/" + p.PodName
}
