package collectors

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/emirozbir/micro-sre/internal/config"
)

type KubernetesCollector struct {
	clientset *kubernetes.Clientset
	config    *config.Config
}

func NewKubernetesCollector(cfg *config.Config) (*KubernetesCollector, error) {
	var k8sConfig *rest.Config
	var err error

	if cfg.Kubernetes.Kubeconfig != "" {
		// Use kubeconfig file
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubernetes.Kubeconfig)
	} else {
		// Use in-cluster config
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			// Fallback to default kubeconfig
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			configOverrides := &clientcmd.ConfigOverrides{}
			if cfg.Kubernetes.Context != "" {
				configOverrides.CurrentContext = cfg.Kubernetes.Context
			}
			k8sConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				loadingRules, configOverrides).ClientConfig()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesCollector{
		clientset: clientset,
		config:    cfg,
	}, nil
}

type PodInfo struct {
	Pod    *corev1.Pod
	Logs   string
	Events []corev1.Event
}

func (k *KubernetesCollector) GetPodInfo(ctx context.Context, namespace, podName string, lookback time.Duration) (*PodInfo, error) {
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	logs, err := k.GetPodLogs(ctx, namespace, podName, lookback)
	if err != nil {
		// Log error but continue
		logs = fmt.Sprintf("Error fetching logs: %v", err)
	}

	events, err := k.GetPodEvents(ctx, namespace, podName, lookback)
	if err != nil {
		// Log error but continue
		events = []corev1.Event{}
	}

	return &PodInfo{
		Pod:    pod,
		Logs:   logs,
		Events: events,
	}, nil
}

func (k *KubernetesCollector) GetPodLogs(ctx context.Context, namespace, podName string, lookback time.Duration) (string, error) {
	sinceTime := metav1.NewTime(time.Now().Add(-lookback))

	opts := &corev1.PodLogOptions{
		SinceTime:  &sinceTime,
		TailLines:  &k.config.LogCollection.TailLines,
		Timestamps: true,
	}

	// Get the main container logs
	req := k.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer podLogs.Close()

	logs, err := io.ReadAll(podLogs)
	if err != nil {
		return "", fmt.Errorf("failed to read pod logs: %w", err)
	}

	return string(logs), nil
}

func (k *KubernetesCollector) GetPodEvents(ctx context.Context, namespace, podName string, lookback time.Duration) ([]corev1.Event, error) {
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", podName)

	eventList, err := k.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Filter events by time
	cutoff := time.Now().Add(-lookback)
	var filteredEvents []corev1.Event
	for _, event := range eventList.Items {
		if event.LastTimestamp.Time.After(cutoff) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents, nil
}

func (k *KubernetesCollector) GetNamespaceEvents(ctx context.Context, namespace string, lookback time.Duration) ([]corev1.Event, error) {
	eventList, err := k.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace events: %w", err)
	}

	cutoff := time.Now().Add(-lookback)
	var filteredEvents []corev1.Event
	for _, event := range eventList.Items {
		if event.LastTimestamp.Time.After(cutoff) {
			// Filter by event type if configured
			if len(k.config.EventCollection.EventTypes) > 0 {
				typeMatch := false
				for _, eventType := range k.config.EventCollection.EventTypes {
					if event.Type == eventType {
						typeMatch = true
						break
					}
				}
				if !typeMatch {
					continue
				}
			}
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents, nil
}

func (k *KubernetesCollector) GetPod(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}
	return pod, nil
}
