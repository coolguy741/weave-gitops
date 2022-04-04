package nsaccess

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typedauth "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// DefautltWegoAppRules is the minimun set of permissions a user will need to use the wego-app in a given namespace
var DefautltWegoAppRules = []rbacv1.PolicyRule{
	{
		APIGroups: []string{""},
		Resources: []string{"secrets", "pods", "events"},
		Verbs:     []string{"get", "list"},
	},
	{
		APIGroups: []string{"apps"},
		Resources: []string{"deployments", "replicasets"},
		Verbs:     []string{"get", "list"},
	},
	{
		APIGroups: []string{"kustomize.toolkit.fluxcd.io"},
		Resources: []string{"kustomizations"},
		Verbs:     []string{"get", "list"},
	},
	{
		APIGroups: []string{"helm.toolkit.fluxcd.io"},
		Resources: []string{"helmreleases"},
		Verbs:     []string{"get", "list"},
	},
	{
		APIGroups: []string{"source.toolkit.fluxcd.io"},
		Resources: []string{"buckets", "helmcharts", "gitrepositories", "helmrepositories"},
		Verbs:     []string{"get", "list"},
	},
	{
		APIGroups: []string{""},
		Resources: []string{"events"},
		Verbs:     []string{"get", "list", "watch"},
	},
}

// Checker contains methods for validing user access to Kubernetes namespaces, based on a set of PolicyRules
//counterfeiter:generate . Checker
type Checker interface {
	// FilterAccessibleNamespaces returns a filtered list of namespaces to which a user has access to
	FilterAccessibleNamespaces(ctx context.Context, cfg *rest.Config, namespaces []corev1.Namespace) ([]corev1.Namespace, error)
}

type simpleChecker struct {
	rules []rbacv1.PolicyRule
}

func NewChecker(rules []rbacv1.PolicyRule) Checker {
	return simpleChecker{rules: rules}
}

func (sc simpleChecker) FilterAccessibleNamespaces(ctx context.Context, cfg *rest.Config, namespaces []corev1.Namespace) ([]corev1.Namespace, error) {
	result := []corev1.Namespace{}

	for _, ns := range namespaces {
		ok, err := userCanUseNamespace(ctx, cfg, ns, sc.rules)
		if err != nil {
			return nil, fmt.Errorf("user namespace access: %w", err)
		}

		if ok {
			result = append(result, ns)
		}
	}

	return result, nil
}

func userCanUseNamespace(ctx context.Context, cfg *rest.Config, ns corev1.Namespace, rules []rbacv1.PolicyRule) (bool, error) {
	auth, err := newAuthClient(cfg)
	if err != nil {
		return false, err
	}

	sar := &authorizationv1.SelfSubjectRulesReview{
		Spec: authorizationv1.SelfSubjectRulesReviewSpec{
			Namespace: ns.Name,
		},
	}

	authRes, err := auth.SelfSubjectRulesReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}

	return hasAllRules(authRes.Status, rules, ns.Name), nil
}

var allK8sVerbs = []string{"create", "get", "list", "watch", "patch", "delete", "deletecollection"}

// hasAll rules determines if a set of SubjectRulesReview rules match a minimum set of policy rules
func hasAllRules(status authorizationv1.SubjectRulesReviewStatus, rules []rbacv1.PolicyRule, ns string) bool {
	hasAccess := true
	// We need to understand the "sum" of all the rules for a role.
	// Convert to a hash lookup to make it easier to tell what a user can do.
	// Looks like { "apps": { "deployments": { get: true, list: true } } }
	derivedAccess := map[string]map[string]map[string]bool{}

	for _, statusRule := range status.ResourceRules {
		for _, apiGroup := range statusRule.APIGroups {
			if _, ok := derivedAccess[apiGroup]; !ok {
				derivedAccess[apiGroup] = map[string]map[string]bool{}
			}

			for _, resource := range statusRule.Resources {
				if _, ok := derivedAccess[apiGroup][resource]; !ok {
					derivedAccess[apiGroup][resource] = map[string]bool{}
				}

				for _, verb := range statusRule.Verbs {
					if verb == "*" {
						for _, v := range allK8sVerbs {
							derivedAccess[apiGroup][resource][v] = true
						}
					} else {
						derivedAccess[apiGroup][resource][verb] = true
					}
				}
			}
		}
	}

Rules:
	for _, rule := range rules {
		for _, apiGroup := range rule.APIGroups {
			g, ok := derivedAccess[apiGroup]

			if !ok {
				hasAccess = false
				continue
			}

		Resources:
			for _, resource := range rule.Resources {
				r, ok := g[resource]
				if !ok {
					// A resource is not present for this apiGroup.
					hasAccess = false
					continue Rules
				}

				for _, verb := range rule.Verbs {
					_, ok := r[verb]
					if !ok {
						// A verb is not present for this resource,
						// no need to check the rest of the verbs.
						hasAccess = false
						continue Resources
					}

					hasAccess = true
				}
			}
		}
	}

	return hasAccess
}

func newAuthClient(cfg *rest.Config) (typedauth.AuthorizationV1Interface, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("making clientset: %w", err)
	}

	return cs.AuthorizationV1(), nil
}
