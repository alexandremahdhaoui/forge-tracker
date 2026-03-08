//go:build e2e

// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// createK8sClient creates a Kubernetes clientset from the given kubeconfig path.
func createK8sClient(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig from %q: %w", kubeconfig, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes clientset: %w", err)
	}

	return clientset, nil
}

// ensureNamespace creates the namespace if it does not already exist.
func ensureNamespace(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("creating namespace %q: %w", namespace, err)
	}

	return nil
}

// createConfigMap creates a ConfigMap in the given namespace with the provided
// YAML content stored under the key "testdata.yaml".
func createConfigMap(ctx context.Context, client *kubernetes.Clientset, namespace, name, data string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"testdata.yaml": data,
		},
	}

	_, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("creating configmap %q in namespace %q: %w", name, namespace, err)
	}

	return nil
}

// createTestRunnerJob creates a Kubernetes Job that runs the test runner image
// with the given ConfigMap mounted at /testdata.
func createTestRunnerJob(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace, name, configMapName, image, serviceURL string,
) error {
	backoffLimit := int32(0)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "testenv-lcr-credentials"},
					},
					Containers: []corev1.Container{
						{
							Name:  "test-runner",
							Image: image,
							Env: []corev1.EnvVar{
								{
									Name:  "SERVICE_URL",
									Value: serviceURL,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "testdata",
									MountPath: "/testdata",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "testdata",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("creating job %q in namespace %q: %w", name, namespace, err)
	}

	return nil
}

// waitForJobCompletion polls the Job status until it completes or fails, or
// the timeout is reached. Returns nil on successful completion, or an error
// if the Job failed or the timeout expired.
func waitForJobCompletion(
	ctx context.Context,
	client *kubernetes.Clientset,
	namespace, name string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		job, err := client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("getting job %q: %w", name, err)
		}

		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
				return nil
			}
			if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
				return fmt.Errorf("job %q failed: %s", name, cond.Message)
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for job %q: %w", name, ctx.Err())
		case <-time.After(pollInterval):
			// Continue polling.
		}
	}

	return fmt.Errorf("timeout waiting for job %q after %v", name, timeout)
}

// getJobPodLogs retrieves the logs from the first pod associated with the given
// Job. It returns the combined log output as a string.
func getJobPodLogs(ctx context.Context, client *kubernetes.Clientset, namespace, jobName string) (string, error) {
	// List pods with the job-name label selector.
	labelSelector := fmt.Sprintf("job-name=%s", jobName)
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("listing pods for job %q: %w", jobName, err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %q", jobName)
	}

	pod := pods.Items[0]
	req := client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("streaming logs from pod %q: %w", pod.Name, err)
	}
	defer stream.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, stream); err != nil {
		return "", fmt.Errorf("reading logs from pod %q: %w", pod.Name, err)
	}

	return buf.String(), nil
}

// cleanupResources deletes all ConfigMaps and Jobs with the given names in the
// specified namespace. Errors are logged but do not cause the function to fail,
// because cleanup runs in a deferred context.
func cleanupResources(ctx context.Context, client *kubernetes.Clientset, namespace string, names []string) {
	propagation := metav1.DeletePropagationBackground

	for _, name := range names {
		// Delete the Job (and its pods via propagation).
		err := client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
			PropagationPolicy: &propagation,
		})
		if err != nil && !errors.IsNotFound(err) {
			fmt.Printf("WARNING: failed to delete job %q: %v\n", name, err)
		}

		// Delete the ConfigMap.
		err = client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			fmt.Printf("WARNING: failed to delete configmap %q: %v\n", name, err)
		}
	}
}

// sanitizeK8sName converts a filename into a valid Kubernetes resource name.
// It lowercases the string, replaces non-alphanumeric characters with hyphens,
// collapses consecutive hyphens, trims leading/trailing hyphens, and truncates
// to 63 characters (the k8s name length limit).
func sanitizeK8sName(name string) string {
	// Remove file extension.
	if idx := strings.LastIndex(name, "."); idx > 0 {
		name = name[:idx]
	}

	name = strings.ToLower(name)

	// Replace non-alphanumeric characters with hyphens.
	re := regexp.MustCompile(`[^a-z0-9]+`)
	name = re.ReplaceAllString(name, "-")

	// Trim leading and trailing hyphens.
	name = strings.Trim(name, "-")

	// Truncate to 63 characters (k8s name length limit).
	if len(name) > 63 {
		name = name[:63]
		// Remove trailing hyphen after truncation.
		name = strings.TrimRight(name, "-")
	}

	return name
}
