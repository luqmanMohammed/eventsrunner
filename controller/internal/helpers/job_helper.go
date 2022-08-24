package helpers

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type jobHelper struct {
	client              client.Client
	listOptions         []client.ListOption
	controllerNamespace string
}

const (
	// JobConditionTypeRunning is a custom condition type when the job is running
	JobConditionTypeRunning batchv1.JobConditionType = "Running"
)

// GetJobState finds the job status based on the job status conditions field.
// go-client doesnt have a job status of Running, to mitigate this a custom
// type has been added in this package
func (jh jobHelper) GetJobStatus(job batchv1.Job) batchv1.JobConditionType {
	for _, condition := range job.Status.Conditions {
		if condition.Status == corev1.ConditionTrue {
			return condition.Type
		}
	}
	return JobConditionTypeRunning
}
