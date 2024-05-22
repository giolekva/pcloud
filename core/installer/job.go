package installer

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/typed/batch/v1"
)

type JobCreator interface {
	Create(name, namespace string, image string, cmd []string) error
}

type noOpJobCreator struct{}

func (c noOpJobCreator) Create(name, namespace string, image string, cmd []string) error {
	return nil
}

func NewNoOpJobCreator() noOpJobCreator {
	return noOpJobCreator{}
}

type realJobCreator struct {
	v1.BatchV1Interface
}

func NewJobCreator(batch v1.BatchV1Interface) *realJobCreator {
	return &realJobCreator{batch}
}

var onFailure corev1.RestartPolicy = "OnFailure"

func (c *realJobCreator) Create(name, namespace string, image string, cmd []string) error {
	_, err := c.Jobs(namespace).Create(context.Background(), &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name:            "job",
							Image:           image,
							ImagePullPolicy: "Always",
							Command:         cmd,
						},
					},
					RestartPolicy: onFailure,
				},
			},
		},
	}, metav1.CreateOptions{})
	return err
}
