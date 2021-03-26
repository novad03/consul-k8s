package connectinject

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// consulSidecar starts the consul-sidecar command to only run
// the metrics merging server when metrics merging feature is enabled.
// It always disables service registration because for connect we no longer
// need to keep services registered as this is handled in the endpoints-controller.
func (h *Handler) consulSidecar(pod corev1.Pod) (corev1.Container, error) {
	run, err := h.shouldRunMergedMetricsServer(pod)
	if err != nil {
		return corev1.Container{}, err
	}

	// This should never happen because we only call this function in the handler if
	// we need to run the metrics merging server. This check is here just in case.
	if !run {
		return corev1.Container{}, errors.New("metrics merging should be enabled in order to inject the consul-sidecar")
	}

	// Configure consul sidecar with the appropriate metrics flags.
	mergedMetricsPort, err := h.mergedMetricsPort(pod)
	if err != nil {
		return corev1.Container{}, err
	}
	serviceMetricsPath := h.serviceMetricsPath(pod)

	// Don't need to check the error since it's checked in the call to
	// h.shouldRunMergedMetricsServer() above.
	serviceMetricsPort, _ := h.serviceMetricsPort(pod)

	command := []string{
		"consul-k8s",
		"consul-sidecar",
		"-enable-service-registration=false",
		"-enable-metrics-merging=true",
		fmt.Sprintf("-merged-metrics-port=%s", mergedMetricsPort),
		fmt.Sprintf("-service-metrics-port=%s", serviceMetricsPort),
		fmt.Sprintf("-service-metrics-path=%s", serviceMetricsPath),
	}

	return corev1.Container{
		Name:  "consul-sidecar",
		Image: h.ImageConsulK8S,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: "/consul/connect-inject",
			},
		},
		Command:   command,
		Resources: h.ConsulSidecarResources,
	}, nil
}
