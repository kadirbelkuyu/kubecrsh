package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type CrashHandler func(crash domain.PodCrash)

type Watcher struct {
	client            kubernetes.Interface
	namespace         string
	factory           informers.SharedInformerFactory
	handler           CrashHandler
	reasons           map[string]bool
	lastNotifications map[string]time.Time
	dedupTTL          time.Duration
	mu                sync.RWMutex
}

type Option func(*Watcher)

func WithNamespace(ns string) Option {
	return func(w *Watcher) {
		w.namespace = ns
	}
}

func WithReasons(reasons []string) Option {
	return func(w *Watcher) {
		for _, r := range reasons {
			w.reasons[r] = true
		}
	}
}

func WithDedupTTL(ttl time.Duration) Option {
	return func(w *Watcher) {
		w.dedupTTL = ttl
	}
}

func New(client kubernetes.Interface, handler CrashHandler, opts ...Option) *Watcher {
	w := &Watcher{
		client:  client,
		handler: handler,
		reasons: map[string]bool{
			"OOMKilled":        true,
			"Error":            true,
			"CrashLoopBackOff": true,
		},
		lastNotifications: make(map[string]time.Time),
		dedupTTL:          5 * time.Minute,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

func (w *Watcher) Start(ctx context.Context) error {
	var factory informers.SharedInformerFactory

	if w.namespace != "" {
		factory = informers.NewSharedInformerFactoryWithOptions(
			w.client,
			0,
			informers.WithNamespace(w.namespace),
		)
	} else {
		factory = informers.NewSharedInformerFactory(w.client, 0)
	}

	w.factory = factory
	podInformer := factory.Core().V1().Pods().Informer()

	_, _ = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return
			}
			w.checkPodOnAdd(pod)
		},
		UpdateFunc: w.onUpdate,
	})

	factory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("failed to sync cache")
	}

	go w.cleanupCacheLoop(ctx)

	<-ctx.Done()
	return nil
}

func (w *Watcher) cleanupCacheLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			now := time.Now()
			for k, t := range w.lastNotifications {
				if now.Sub(t) > w.dedupTTL*2 {
					delete(w.lastNotifications, k)
				}
			}
			w.mu.Unlock()
		}
	}
}

func (w *Watcher) onUpdate(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*corev1.Pod)
	if !ok {
		return
	}
	newPod, ok := newObj.(*corev1.Pod)
	if !ok {
		return
	}

	w.detectCrashes(oldPod, newPod)
}

func (w *Watcher) detectCrashes(oldPod, newPod *corev1.Pod) {
	for i, cs := range newPod.Status.ContainerStatuses {
		var oldStatus *corev1.ContainerStatus
		if i < len(oldPod.Status.ContainerStatuses) {
			oldStatus = &oldPod.Status.ContainerStatuses[i]
		}

		if crash := w.checkContainerCrash(newPod, cs, oldStatus); crash != nil {
			if w.shouldNotify(crash) {
				w.handler(*crash)
			}
		}
	}
}

func (w *Watcher) checkContainerCrash(pod *corev1.Pod, cs corev1.ContainerStatus, oldStatus *corev1.ContainerStatus) *domain.PodCrash {
	if cs.State.Terminated != nil {
		if oldStatus == nil || oldStatus.State.Terminated == nil {
			return w.createCrashFromTerminated(pod, cs)
		}
	}

	if cs.LastTerminationState.Terminated != nil {
		if oldStatus == nil ||
			oldStatus.LastTerminationState.Terminated == nil ||
			cs.RestartCount > oldStatus.RestartCount {
			return w.createCrashFromLastTerminated(pod, cs)
		}
	}

	if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
		if oldStatus == nil ||
			oldStatus.State.Waiting == nil ||
			oldStatus.State.Waiting.Reason != "CrashLoopBackOff" {
			return w.createCrashLoopBackOff(pod, cs)
		}
	}

	return nil
}

func (w *Watcher) createCrashFromTerminated(pod *corev1.Pod, cs corev1.ContainerStatus) *domain.PodCrash {
	terminated := cs.State.Terminated
	reason := terminated.Reason
	if reason == "" {
		reason = "Error"
	}

	if !w.shouldHandle(reason) {
		return nil
	}

	return &domain.PodCrash{
		Namespace:     pod.Namespace,
		PodName:       pod.Name,
		ContainerName: cs.Name,
		ExitCode:      terminated.ExitCode,
		Reason:        reason,
		Signal:        terminated.Signal,
		RestartCount:  cs.RestartCount,
		StartedAt:     terminated.StartedAt.Time,
		FinishedAt:    terminated.FinishedAt.Time,
	}
}

func (w *Watcher) createCrashFromLastTerminated(pod *corev1.Pod, cs corev1.ContainerStatus) *domain.PodCrash {
	terminated := cs.LastTerminationState.Terminated
	reason := terminated.Reason
	if reason == "" {
		reason = "Error"
	}

	if !w.shouldHandle(reason) {
		return nil
	}

	return &domain.PodCrash{
		Namespace:     pod.Namespace,
		PodName:       pod.Name,
		ContainerName: cs.Name,
		ExitCode:      terminated.ExitCode,
		Reason:        reason,
		Signal:        terminated.Signal,
		RestartCount:  cs.RestartCount,
		StartedAt:     terminated.StartedAt.Time,
		FinishedAt:    terminated.FinishedAt.Time,
	}
}

func (w *Watcher) createCrashLoopBackOff(pod *corev1.Pod, cs corev1.ContainerStatus) *domain.PodCrash {
	if !w.shouldHandle("CrashLoopBackOff") {
		return nil
	}

	crash := &domain.PodCrash{
		Namespace:     pod.Namespace,
		PodName:       pod.Name,
		ContainerName: cs.Name,
		Reason:        "CrashLoopBackOff",
		RestartCount:  cs.RestartCount,
	}

	if cs.LastTerminationState.Terminated != nil {
		crash.ExitCode = cs.LastTerminationState.Terminated.ExitCode
		crash.Signal = cs.LastTerminationState.Terminated.Signal
		crash.StartedAt = cs.LastTerminationState.Terminated.StartedAt.Time
		crash.FinishedAt = cs.LastTerminationState.Terminated.FinishedAt.Time
	}

	return crash
}

func (w *Watcher) shouldHandle(reason string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.reasons[reason]
}

func (w *Watcher) shouldNotify(crash *domain.PodCrash) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := fmt.Sprintf("%s/%s/%s/%s", crash.Namespace, crash.PodName, crash.ContainerName, crash.Reason)

	lastTime, exists := w.lastNotifications[key]
	if exists && time.Since(lastTime) < w.dedupTTL {
		return false
	}

	w.lastNotifications[key] = time.Now()
	return true
}

func (w *Watcher) checkPodOnAdd(pod *corev1.Pod) {
	for _, cs := range pod.Status.ContainerStatuses {
		if crash := w.checkContainerCrash(pod, cs, nil); crash != nil {
			if w.shouldNotify(crash) {
				w.handler(*crash)
			}
		}
	}
}
