package framework

import (
	"fmt"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"github.com/pkg/errors"
	"net/url"
	"time"
)

func (i *lbInvocation) CreateService(selector map[string]string) error {
	_, err := i.kubeClient.CoreV1().Services(i.Namespace()).Create(&core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServerResourceName,
			Namespace: i.Namespace(),
			Labels: map[string]string{
				"app": "test-server-" + i.app,
			},
		},
		Spec: core.ServiceSpec{
			Ports: i.testServerServicePorts(),
			Selector: selector,
			Type: core.ServiceTypeLoadBalancer,
		},
	})

	return err
}

func (i *lbInvocation) getLoadBalancerURLs() ([]string, error) {
	var serverAddr []string

	svc, err := i.GetServiceWithLoadBalancerStatus(testServerResourceName, i.Namespace())
	if err != nil {
		return  serverAddr, err
	}

	ips := make([]string, 0)
	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		ips = append(ips, ingress.IP)
	}

	var ports []int32
	if len(svc.Spec.Ports) > 0 {
		for _, port := range svc.Spec.Ports {
			if port.NodePort > 0 {
				ports = append(ports, port.Port)
			}
		}
	}

	for _, port := range ports {
		for _, ip := range ips {
			u, err := url.Parse(fmt.Sprintf("http://%s:%d", ip, port))
			if err != nil {
				return nil, err
			}
			serverAddr = append(serverAddr, u.String())
		}
	}

	return serverAddr, nil
}

func (i *lbInvocation) GetServiceWithLoadBalancerStatus(name, namespace string) (*core.Service, error) {
	var (
		svc *core.Service
		err error
	)
	err = wait.PollImmediate(2*time.Second, 20*time.Minute, func() (bool, error) {
		svc, err = i.kubeClient.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		if err != nil || len(svc.Status.LoadBalancer.Ingress) == 0 { // retry
			return false, nil
		} else {
			return true, nil
		}
	})
	if err != nil {
		return nil, errors.Errorf("failed to get Status.LoadBalancer.Ingress for service %s/%s", name, namespace)
	}
	return svc, nil
}


func (i *lbInvocation) testServerServicePorts() []core.ServicePort {
	return []core.ServicePort{
		{
			Name:       "http-1",
			Port:       80,
			TargetPort: intstr.FromInt(8080),
			Protocol:   "TCP",
		},
	}
}