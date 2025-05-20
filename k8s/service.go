package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeServices(serviceName string, numPods int) []*corev1.Service {
	ss := make([]*corev1.Service, numPods)
	for i := range ss {
		ss[i] = MakeService(serviceName, i)
	}
	return ss
}

func MakeService(serviceName string, podIndex int) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%d", serviceName, podIndex+1),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":                                serviceName,
				"statefulset.kubernetes.io/pod-name": fmt.Sprintf("%s-%d", serviceName, podIndex),
			},
			Ports: []corev1.ServicePort{
				{
					Port: 9111,
					Name: "app",
				},
				{
					Port: 2222,
					Name: "metrics",
				},
			},
		},
	}
}
