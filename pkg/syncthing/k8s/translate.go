package k8s

import (
	"fmt"
	"path/filepath"

	"github.com/okteto/okteto/pkg/k8s/secrets"
	"github.com/okteto/okteto/pkg/model"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	oktetoSyncLabel     = "syncthing.okteto.com"
	syncImageTag        = "okteto/syncthing:1.1.4"
	syncTCPPort         = 22000
	syncGUIPort         = 8384
	oktetoContainer     = "okteto"
	oktetoSecretVolume  = "okteto-secret"
	oktetoInitContainer = "okteto-init"
	oktetoMount         = "/var/okteto"
)

var (
	devReplicas int32 = 1
)

func translate(dev *model.Dev) *appsv1.StatefulSet {
	initContainer := translateInitContainer(dev)

	reqMem, _ := resource.ParseQuantity("64Mi")
	reqCPU, _ := resource.ParseQuantity("50m")
	limMem, _ := resource.ParseQuantity("256Mi")
	limCPU, _ := resource.ParseQuantity("200m")
	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dev.GetSyncStatefulSetName(),
			Namespace: dev.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: dev.GetSyncStatefulSetName(),
			Replicas:    &devReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					oktetoSyncLabel: dev.Name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						oktetoSyncLabel: dev.Name,
					},
				},
				Spec: apiv1.PodSpec{
					TerminationGracePeriodSeconds: &devTerminationGracePeriodSeconds,
					InitContainers:                []apiv1.Container{*initContainer},
					Containers: []apiv1.Container{
						apiv1.Container{
							Name:            oktetoContainer,
							Image:           syncImageTag,
							ImagePullPolicy: apiv1.PullIfNotPresent,
							Resources: apiv1.ResourceRequirements{
								Requests: apiv1.ResourceList{
									apiv1.ResourceMemory: reqMem,
									apiv1.ResourceCPU:    reqCPU,
								},
								Limits: apiv1.ResourceList{
									apiv1.ResourceMemory: limMem,
									apiv1.ResourceCPU:    limCPU,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								apiv1.VolumeMount{
									Name:      oktetoSecretVolume,
									MountPath: "/var/syncthing/secret/",
								},
								apiv1.VolumeMount{
									Name:      dev.GetSyncVolumeName(),
									MountPath: oktetoMount,
								},
							},
							Ports: []apiv1.ContainerPort{
								apiv1.ContainerPort{
									ContainerPort: syncGUIPort,
								},
								apiv1.ContainerPort{
									ContainerPort: syncTCPPort,
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						apiv1.Volume{
							Name: oktetoSecretVolume,
							VolumeSource: apiv1.VolumeSource{
								Secret: &apiv1.SecretVolumeSource{
									SecretName: secrets.GetSecretName(dev),
								},
							},
						},
					},
				},
			},
		},
	}
	AddCodeVolume(dev, &ss.Spec.Template.Spec)

	return ss
}

func translateInitContainer(dev *model.Dev) *apiv1.Container {
	reqMem, _ := resource.ParseQuantity("16Mi")
	reqCPU, _ := resource.ParseQuantity("50m")
	limMem, _ := resource.ParseQuantity("16Mi")
	limCPU, _ := resource.ParseQuantity("50m")
	source := filepath.Join(dev.WorkDir, "*")

	return &apiv1.Container{
		Name:    oktetoInitContainer,
		Image:   dev.Image,
		Command: []string{"sh", "-c", fmt.Sprintf("(ls -A /okteto/init | grep -v lost+found || cp -Rf %s /okteto/init); touch /okteto/init/%s", source, dev.DevPath)},
		Resources: apiv1.ResourceRequirements{
			Requests: apiv1.ResourceList{
				apiv1.ResourceMemory: reqMem,
				apiv1.ResourceCPU:    reqCPU,
			},
			Limits: apiv1.ResourceList{
				apiv1.ResourceMemory: limMem,
				apiv1.ResourceCPU:    limCPU,
			},
		},
		VolumeMounts: []apiv1.VolumeMount{
			apiv1.VolumeMount{
				Name:      dev.GetSyncVolumeName(),
				MountPath: "/okteto/init",
			},
		},
	}
}

//AddCodeVolume adds the code volume info to a pod spec
func AddCodeVolume(dev *model.Dev, spec *apiv1.PodSpec) {
	if spec.Volumes == nil {
		spec.Volumes = []apiv1.Volume{}
	}
	for _, v := range spec.Volumes {
		if v.Name == dev.GetSyncVolumeName() {
			return
		}
	}
	v := apiv1.Volume{
		Name: dev.GetSyncVolumeName(),
		VolumeSource: apiv1.VolumeSource{
			PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
				ClaimName: dev.GetSyncVolumeName(),
				ReadOnly:  false,
			},
		},
	}
	spec.Volumes = append(spec.Volumes, v)
}