package permission

import (
	"fmt"
	"log"
	"strings"

	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// ByID check if a permissions exist, and return that permissions
func ByID(client *govmomi.Client, id string) (*types.Permission, error) {
	log.Printf("[DEBUG] Locating entity permission with ID %s", id)
	entityID, entityType, principal, err := SplitID(id)
	if err != nil {
		return nil, err
	}
	m := object.NewAuthorizationManager(client.Client)
	finder := find.NewFinder(client.Client, true)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	ref := types.ManagedObjectReference{
		Type:  entityType,
		Value: entityID,
	}

	entity, err := finder.ObjectReference(ctx, ref)
	if err != nil {
		return nil, err
	}

	permissions, err := m.RetrieveEntityPermissions(ctx, entity.Reference(), true)
	if err != nil {
		return nil, err
	}

	for _, permission := range permissions {
		if permission.Principal == principal {
			log.Printf("[DEBUG] Found entity permission with ID %s", id)
			return &permission, nil
		}
	}

	return nil, fmt.Errorf("no prinicipal with name %q for entity permission %q", principal, id)
}

// Create Entity Permission
func Create(client *govmomi.Client, entityID string, entityType string, principal string, roleID int, group bool, propagate bool) error {
	log.Printf("[DEBUG] Creating entity permission %s:%s:%d", entityID, principal, roleID)
	m := object.NewAuthorizationManager(client.Client)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	ref := types.ManagedObjectReference{
		Type:  entityType,
		Value: entityID,
	}

	perms := []types.Permission{types.Permission{
		Principal: principal,
		RoleId:    int32(roleID),
		Group:     group,
		Propagate: propagate,
	}}

	if err := m.SetEntityPermissions(ctx, ref, perms); err != nil {
		return err
	}
	log.Printf("[DEBUG] Successfully created entity permission %s:%s:%d", entityID, principal, roleID)
	return nil
}

// Remove Entity Permission
func Remove(client *govmomi.Client, permission *types.Permission) error {
	log.Printf("[DEBUG] Removing entity permission %s:%s:%d", permission.Entity.Value, permission.Principal, permission.RoleId)
	m := object.NewAuthorizationManager(client.Client)
	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	if err := m.RemoveEntityPermission(ctx, permission.Entity.Reference(), permission.Principal, permission.Group); err != nil {
		return err
	}
	log.Printf("[DEBUG] Removing entity permission %s:%s:%d", permission.Entity.Value, permission.Principal, permission.RoleId)
	return nil
}

// SplitID takes the permission's ID and splits it into the folder and principal.
func SplitID(id string) (string, string, string, error) {
	s := strings.Split(id, ":")
	if len(s) != 3 {
		return "", "", "", fmt.Errorf("role ID does not contain principal, entity type, and entity ID")
	}
	return s[0], s[1], s[2], nil
}

// ConcatID takes a permission's folder and principal and generates an ID.
func ConcatID(id, entityType, principal string) string {
	return fmt.Sprintf("%s:%s:%s", id, entityType, principal)
}
