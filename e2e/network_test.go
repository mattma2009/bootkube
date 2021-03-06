package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func TestNetwork(t *testing.T) {
	//
	// 1. create nginx service
	di, _, err := api.Codecs.UniversalDecoder().Decode(nginxDepNT, nil, &v1beta1.Deployment{})
	if err != nil {
		t.Fatalf("unable to decode deployment manifest: %v", err)
	}

	d, ok := di.(*v1beta1.Deployment)
	if !ok {
		t.Fatalf("expected manifest to decode into *api.deployment, got %T", di)
	}
	_, err = client.ExtensionsV1beta1().Deployments(namespace).Create(d)
	if err != nil {
		t.Fatal(err)
	}

	deleteDeployment := func() {
		delPropPolicy := metav1.DeletePropagationForeground
		client.ExtensionsV1beta1().Deployments(namespace).Delete("nginx-deployment-nt", &metav1.DeleteOptions{
			PropagationPolicy: &delPropPolicy,
		})
	}
	defer deleteDeployment()

	if err := retry(10, time.Second*10, getNginxPod); err != nil {
		t.Fatalf("timed out waiting for nginx pod: %v", err)
	}

	si, _, err := api.Codecs.UniversalDecoder().Decode(nginxSVCNT, nil, &v1.Service{})
	if err != nil {
		t.Fatalf("unable to decode service manifest: %v", err)
	}
	s, ok := si.(*v1.Service)
	if !ok {
		t.Fatalf("expected manifest to decode into *api.service, got %T", si)
	}
	_, err = client.CoreV1().Services(namespace).Create(s)
	if err != nil {
		t.Fatal(err)
	}
	defer client.CoreV1().Services(namespace).Delete("nginx-service-nt", &metav1.DeleteOptions{})

	//
	// 2. create a wget pod that hits the nginx service
	testPodName := fmt.Sprintf("%s-%s", "wget-pod-nt", utilrand.String(5))
	wgetPodNT.ObjectMeta.Name = testPodName
	_, err = client.CoreV1().Pods(namespace).Create(wgetPodNT)
	if err != nil {
		t.Fatal(err)
	}

	if err := retry(10, time.Second*10, getPod(testPodName)); err != nil {
		t.Fatalf(fmt.Sprintf("timed out waiting for wget pod to succeed: %v", err))
	}

	t.Run("DefaultDeny", HelperDefaultDeny)
	t.Run("NetworkPolicy", HelperPolicy)
}

func HelperDefaultDeny(t *testing.T) {
	//
	// 3. set DefaultDeny policy
	npi, _, err := api.Codecs.UniversalDecoder().Decode(defaultDenyNetworkPolicy, nil, &v1beta1.NetworkPolicy{})
	if err != nil {
		t.Fatalf("unable to decode network policy manifest: %v", err)
	}

	np, ok := npi.(*v1beta1.NetworkPolicy)
	if !ok {
		t.Fatalf("expected manifest to decode into *api.networkpolicy, got %T", npi)
	}

	httpRestClient := client.ExtensionsV1beta1().RESTClient()
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s",
		strings.ToLower("extensions"),
		strings.ToLower("v1beta1"),
		strings.ToLower(namespace),
		strings.ToLower("NetworkPolicies"))

	result := httpRestClient.Post().RequestURI(uri).Body(np).Do()
	if result.Error() != nil {
		t.Fatal(result.Error())
	}
	defer func() {
		uri = fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s",
			strings.ToLower("extensions"),
			strings.ToLower("v1beta1"),
			strings.ToLower(namespace),
			strings.ToLower("NetworkPolicies"),
			strings.ToLower(np.ObjectMeta.Name))

		result = httpRestClient.Delete().RequestURI(uri).Do()
		if result.Error() != nil {
			t.Fatal(result.Error())
		}

	}()

	//
	// 4. create a wget pod that fails to hit nginx service
	testPodName := fmt.Sprintf("%s-%s", "wget-pod-nt", utilrand.String(5))
	wgetPodNT.ObjectMeta.Name = testPodName
	_, err = client.CoreV1().Pods(namespace).Create(wgetPodNT)
	if err != nil {
		t.Fatal(err)
	}

	if err := retry(10, time.Second*10, getFailedPod(testPodName)); err != nil {
		t.Fatalf(fmt.Sprintf("timed out waiting for wget pod to fail: %v", err))
	}
}

func HelperPolicy(t *testing.T) {
	//
	// 5. create NetworkPolicy that allows `allow=access`
	npi, _, err := api.Codecs.UniversalDecoder().Decode(netPolicy, nil, &v1beta1.NetworkPolicy{})
	if err != nil {
		t.Fatalf("unable to decode network policy manifest: %v", err)
	}

	np, ok := npi.(*v1beta1.NetworkPolicy)
	if !ok {
		t.Fatalf("expected manifest to decode into *api.networkpolicy, got %T", npi)
	}

	httpRestClient := client.ExtensionsV1beta1().RESTClient()
	uri := fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s",
		strings.ToLower("extensions"),
		strings.ToLower("v1beta1"),
		strings.ToLower(namespace),
		strings.ToLower("NetworkPolicies"))

	result := httpRestClient.Post().RequestURI(uri).Body(np).Do()
	if result.Error() != nil {
		t.Fatal(result.Error())
	}
	defer func() {
		uri = fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s",
			strings.ToLower("extensions"),
			strings.ToLower("v1beta1"),
			strings.ToLower(namespace),
			strings.ToLower("NetworkPolicies"),
			strings.ToLower(np.ObjectMeta.Name))

		result = httpRestClient.Delete().RequestURI(uri).Do()
		if result.Error() != nil {
			t.Fatal(result.Error())
		}

	}()

	//
	// 6. create a wget pod with label `allow=access` that hits the nginx service
	testPodName := fmt.Sprintf("%s-%s", "wget-pod-nt", utilrand.String(5))
	wgetPodNT.ObjectMeta.Name = testPodName
	wgetPodNT.ObjectMeta.Labels = map[string]string{}
	wgetPodNT.ObjectMeta.Labels["allow"] = "access"
	_, err = client.CoreV1().Pods(namespace).Create(wgetPodNT)
	if err != nil {
		t.Fatal(err)
	}

	if err := retry(10, time.Second*10, getPod(testPodName)); err != nil {
		t.Fatalf(fmt.Sprintf("timed out waiting for wget pod to succeed: %v", err))
	}

	//
	// 7. create a wget pod with label `allow=cant-access` that fails to the nginx service
	testPodName = fmt.Sprintf("%s-%s", "wget-pod-nt", utilrand.String(5))
	wgetPodNT.ObjectMeta.Name = testPodName
	wgetPodNT.ObjectMeta.Labels["allow"] = "cant-access"
	_, err = client.CoreV1().Pods(namespace).Create(wgetPodNT)
	if err != nil {
		t.Fatal(err)
	}

	if err := retry(10, time.Second*10, getFailedPod(testPodName)); err != nil {
		t.Fatalf(fmt.Sprintf("timed out waiting for wget pod to fail: %v", err))
	}
}

func getNginxPod() error {
	l, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "app=nginx"})
	if err != nil || len(l.Items) == 0 {
		return fmt.Errorf("couldn't list pods: %v", err)
	}

	// take the first pod
	p := &l.Items[0]

	if p.Status.Phase != v1.PodRunning {
		return fmt.Errorf("pod not yet running: %v", p.Status.Phase)
	}
	return nil
}

func getPod(name string) func() error {
	return func() error {
		p, err := client.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("couldn't get pod: %v", err)
		}
		if p.Status.Phase != v1.PodSucceeded {
			return fmt.Errorf("pod did not succeed: %v", p.Status.Phase)
		}
		return nil
	}
}

func getFailedPod(name string) func() error {
	return func() error {
		p, err := client.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("couldn't get pod: %v", err)
		}
		if p.Status.Phase != v1.PodFailed {
			return fmt.Errorf("pod did not fail: %v", p.Status.Phase)
		}
		return nil
	}
}

var nginxDepNT = []byte(`apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: nginx-deployment-nt
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.8
        ports:
        - containerPort: 80
`)

var wgetPodNT = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: namespace,
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:    "wget-container",
				Image:   "busybox:1.26",
				Command: []string{"wget", "--timeout", "5", "nginx-service-nt"},
			},
		},
		RestartPolicy: v1.RestartPolicyNever,
	},
}

var nginxSVCNT = []byte(`apiVersion: v1
kind: Service
metadata:
  name: nginx-service-nt
spec:
  selector:
    app: nginx
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
`)

var defaultDenyNetworkPolicy = []byte(`kind: NetworkPolicy                                                      
apiVersion: extensions/v1beta1
metadata:
  name: default-deny
spec:
  podSelector:
`)

var netPolicy = []byte(`kind: NetworkPolicy                                                      
apiVersion: extensions/v1beta1
metadata:
  name: access-nginx
spec:
  podSelector:
    matchLabels:
      app: nginx
  ingress:
    - from:
      - podSelector:
          matchLabels:
            allow: access
`)
