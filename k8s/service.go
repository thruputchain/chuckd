package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeServices(serviceName string, numPods int) ([]*corev1.Service, []string) {
	ss := make([]*corev1.Service, numPods)
	addrs := make([]string, numPods)
	for i := range ss {
		dnsName := fmt.Sprintf("%s-%d", serviceName, i+1)
		ss[i] = MakeService(serviceName, i, dnsName)
		addrs[i] = fmt.Sprintf("%s:%d", dnsName, 9111)
	}
	return ss, addrs
}

func MakeService(serviceName string, podIndex int, dnsName string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: dnsName,
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
