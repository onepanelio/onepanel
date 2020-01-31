package kube

import "k8s.io/client-go/kubernetes/fake"

func NewTestClient() (client *Client) {
	return &Client{Interface: fake.NewSimpleClientset()}
}
