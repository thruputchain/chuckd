package k8s

import (
	"strings"

	proto "github.com/groundhogchain/protocol-go"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func MakeStatefulSet(
	serviceName string,
	containerImage string,
	subcmd string,
	cfg []proto.Config,
	services []*v1.Service,
) []runtime.Object {

	cfgMaps, cfgVolumes, cfgMounts := MakeConfigMapsForStatefulSet(serviceName, cfg)

	statefulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: serviceName,
			Replicas:    int32Ptr(int32(len(services))),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": serviceName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": serviceName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            serviceName,
							Image:           containerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/bin/sh",
								"-c",
								strings.Join([]string{
									"source /app/config/env.sh", // exports INSTANCE and SEED, prepared by InitContainers
									"exec /app/chuckd --config=/app/config/config.json --instance $INSTANCE --seed $SEED " + subcmd,
								}, "\n"),
							},
							Env: []corev1.EnvVar{
								{
									Name: "INSTANCE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config-volume",
									MountPath: "/app/config",
								},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9111, Name: "chuck"},
								{ContainerPort: 6666, Name: "wallet"},
								{ContainerPort: 2222, Name: "metrics"},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "config-setup",
							Image: "busybox",
							Command: []string{
								"sh",
								"-c",
								strings.Join([]string{
									"POD_NAME=$(echo $HOSTNAME | awk -F- '{print $NF}')",
									"POD_INDEX=$((POD_NAME+1))",
									"cp /configs/config-$POD_INDEX/config.json /target/config.json",
									"SEED=$((POD_INDEX * 23))",
									"echo \"export SEED=$SEED\" > /target/env.sh",
									"echo \"export INSTANCE=" + serviceName + "-$POD_INDEX\" >> /target/env.sh",
								}, "\n"),
							},
							VolumeMounts: append(
								[]corev1.VolumeMount{{
									Name:      "config-volume",
									MountPath: "/target",
								}}, cfgMounts...),
						},
					},
					Volumes: append(
						[]corev1.Volume{{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						}}, cfgVolumes...),
				},
			},
		},
	}

	resources := []runtime.Object{}
	for _, cfgMap := range cfgMaps {
		resources = append(resources, cfgMap)
	}
	for _, service := range services {
		resources = append(resources, service)
	}
	resources = append(resources, statefulSet)

	return resources
}
