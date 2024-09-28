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

// UserRolesHandler handles requests to show the roles or cluster roles a user has access to.
func UserRolesHandler(clientset *kubernetes.Clientset) echo.HandlerFunc {
	return func(c echo.Context) error {
		username := c.Get("username").(string)
		isAdmin, ok := c.Get("isAdmin").(bool)
		if !ok {
			return echo.NewHTTPError(http.StatusForbidden, "Unable to determine admin status")
		}

		if !isAdmin && !auth.HasPermission(username, "view_user_roles") {
			return echo.NewHTTPError(http.StatusForbidden, "You do not have permission to view user roles")
		}

		userName := c.QueryParam("userName")
		if userName == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "User name is required")
		}

		roleBindings, err := clientset.RbacV1().RoleBindings("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return utils.LogAndRespondError(c, http.StatusInternalServerError, "Error listing role bindings", err, "Failed to list role bindings")
		}

		clusterRoleBindings, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return utils.LogAndRespondError(c, http.StatusInternalServerError, "Error listing cluster role bindings", err, "Failed to list cluster role bindings")
		}

		userRoles := extractUserRoles(userName, roleBindings.Items, clusterRoleBindings.Items)
		return c.JSON(http.StatusOK, userRoles)
	}
}

// extractUserRoles extracts the roles and cluster roles a user has access to.
func extractUserRoles(username string, roleBindings []rbacv1.RoleBinding, clusterRoleBindings []rbacv1.ClusterRoleBinding) []string {
	var roles []string

	for _, rb := range roleBindings {
		for _, subject := range rb.Subjects {
			if subject.Kind == rbacv1.UserKind && subject.Name == username {
				roles = append(roles, rb.RoleRef.Name)
			}
		}
	}

	for _, crb := range clusterRoleBindings {
		for _, subject := range crb.Subjects {
			if subject.Kind == rbacv1.UserKind && subject.Name == username {
				roles = append(roles, crb.RoleRef.Name)
			}
		}
	}

	return roles
}