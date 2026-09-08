package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/kadirbelkuyu/kubecrsh/internal/domain"
)

type CrashHandler func(crash domain.PodCrash)

type Watcher struct {
	client      kubernetes.Interface
	namespace   string
	factory     informers.SharedInformerFactory
	handler     CrashHandler
	reasons     map[string]bool
	crashStates map[containerKey]crashState
	dedupTTL    time.Duration
	mu          sync.RWMutex
}

type containerKey struct {
	podIdentity   string
	containerName string
}

type crashState struct {
	readySince time.Time
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
		if ttl > 0 {
			w.dedupTTL = ttl
		}
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
		crashStates: make(map[containerKey]crashState),
		dedupTTL:    5 * time.Minute,
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
		DeleteFunc: w.onDelete,
	})

	factory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("failed to sync cache")
	}

	<-ctx.Done()
	return nil
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
			if w.shouldNotify(newPod, cs.Name) {
				w.handler(*crash)
			}
			continue
		}

		w.observeContainer(newPod, cs)
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

func (w *Watcher) shouldNotify(pod *corev1.Pod, containerName string) bool {
	key := containerKey{
		podIdentity:   podIdentity(pod),
		containerName: containerName,
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	state, exists := w.crashStates[key]
	now := time.Now()
	if exists && !state.readySince.IsZero() && now.Sub(state.readySince) >= w.dedupTTL {
		delete(w.crashStates, key)
		exists = false
	}

	w.crashStates[key] = crashState{}
	return !exists
}

func (w *Watcher) checkPodOnAdd(pod *corev1.Pod) {
	for _, cs := range pod.Status.ContainerStatuses {
		if crash := w.checkContainerCrash(pod, cs, nil); crash != nil {
			if w.shouldNotify(pod, cs.Name) {
				w.handler(*crash)
			}
			continue
		}

		w.observeContainer(pod, cs)
	}
}

func (w *Watcher) observeContainer(pod *corev1.Pod, cs corev1.ContainerStatus) {
	key := containerKey{
		podIdentity:   podIdentity(pod),
		containerName: cs.Name,
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	state, exists := w.crashStates[key]
	if !exists {
		return
	}

	if cs.State.Running == nil || !cs.Ready {
		state.readySince = time.Time{}
		w.crashStates[key] = state
		return
	}

	if state.readySince.IsZero() {
		state.readySince = time.Now()
		w.crashStates[key] = state
		return
	}

	if time.Since(state.readySince) >= w.dedupTTL {
		delete(w.crashStates, key)
		return
	}

	w.crashStates[key] = state
}

func (w *Watcher) onDelete(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		pod, _ = tombstone.Obj.(*corev1.Pod)
	}
	if pod == nil {
		return
	}

	identity := podIdentity(pod)
	w.mu.Lock()
	defer w.mu.Unlock()
	for key := range w.crashStates {
		if key.podIdentity == identity {
			delete(w.crashStates, key)
		}
	}
}

func podIdentity(pod *corev1.Pod) string {
	if pod.UID != "" {
		return string(pod.UID)
	}
	return fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
}
