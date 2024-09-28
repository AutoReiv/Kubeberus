package rbac

import (
	"context"
	"net/http"

	"rbac/pkg/auth"
	"rbac/pkg/utils"

	"github.com/labstack/echo/v4"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ServiceAccountDetailsResponse represents the detailed information about a service account.
type ServiceAccountDetailsResponse struct {
	ServiceAccountName  string                      `json:"serviceAccountName"`
	RoleBindings        []rbacv1.RoleBinding        `json:"roleBindings"`
	ClusterRoleBindings []rbacv1.ClusterRoleBinding `json:"clusterRoleBindings"`
	ClusterRoles        []rbacv1.ClusterRole        `json:"clusterRoles"`
}

// ServiceAccountDetailsHandler handles requests for detailed information about a specific service account.
func ServiceAccountDetailsHandler(clientset *kubernetes.Clientset) echo.HandlerFunc {
	return func(c echo.Context) error {
		username := c.Get("username").(string)
		isAdmin, ok := c.Get("isAdmin").(bool)
		if !ok {
			return echo.NewHTTPError(http.StatusForbidden, "Unable to determine admin status")
		}

		if !isAdmin && !auth.HasPermission(username, "view_serviceaccount_details") {
			return echo.NewHTTPError(http.StatusForbidden, "You do not have permission to view service account details")
		}

		serviceAccountName := c.QueryParam("serviceAccountName")
		if serviceAccountName == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Service account name is required")
		}

		roleBindings, err := clientset.RbacV1().RoleBindings("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return utils.LogAndRespondError(c, http.StatusInternalServerError, "Error listing role bindings", err, "Failed to list role bindings")
		}

		clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return utils.LogAndRespondError(c, http.StatusInternalServerError, "Error listing cluster role bindings", err, "Failed to list cluster role bindings")
		}

		clusterRoles, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return utils.LogAndRespondError(c, http.StatusInternalServerError, "Error listing cluster roles", err, "Failed to list cluster roles")
		}

		serviceAccountDetails := extractServiceAccountDetails(serviceAccountName, roleBindings.Items, clusterRoleBindings.Items, clusterRoles.Items)
		return c.JSON(http.StatusOK, serviceAccountDetails)
	}
}

// extractServiceAccountDetails extracts detailed information about a specific service account.
func extractServiceAccountDetails(serviceAccountName string, roleBindings []rbacv1.RoleBinding, clusterRoleBindings []rbacv1.ClusterRoleBinding, clusterRoles []rbacv1.ClusterRole) ServiceAccountDetailsResponse {
	var serviceAccountRoleBindings []rbacv1.RoleBinding
	var serviceAccountClusterRoleBindings []rbacv1.ClusterRoleBinding
	var serviceAccountClusterRoles []rbacv1.ClusterRole

	for _, rb := range roleBindings {
		for _, subject := range rb.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind && subject.Name == serviceAccountName {
				serviceAccountRoleBindings = append(serviceAccountRoleBindings, rb)
			}
		}
	}

	for _, crb := range clusterRoleBindings {
		for _, subject := range crb.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind && subject.Name == serviceAccountName {
				serviceAccountClusterRoleBindings = append(serviceAccountClusterRoleBindings, crb)
			}
		}
	}

	// Collect ClusterRoles associated with the service account's ClusterRoleBindings
	for _, crb := range serviceAccountClusterRoleBindings {
		for _, cr := range clusterRoles {
			if cr.Name == crb.RoleRef.Name {
				serviceAccountClusterRoles = append(serviceAccountClusterRoles, cr)
			}
		}
	}

	return ServiceAccountDetailsResponse{
		ServiceAccountName:  serviceAccountName,
		RoleBindings:        serviceAccountRoleBindings,
		ClusterRoleBindings: serviceAccountClusterRoleBindings,
		ClusterRoles:        serviceAccountClusterRoles,
	}
}
