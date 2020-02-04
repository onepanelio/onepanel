package kube

import (
	authorizationv1 "k8s.io/api/authorization/v1"
)

func (c *Client) IsAuthorized(namespace, verb, group, resource, name string) (allowed bool, err error) {
	review, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(&authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     group,
				Resource:  resource,
				Name:      name,
			},
		},
	})
	if err != nil {
		allowed = false
		return
	}
	allowed = review.Status.Allowed

	return
}
