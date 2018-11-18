package role

import (
	"errors"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// Exists check if a permissions exist, and return that permissions
func Exists(client *govmomi.Client, principal string, folderPath string) (*types.Permission, error) {
	m := object.NewAuthorizationManager(client.Client)
	finder := find.NewFinder(client.Client, true)
	ctx, cancel := context.WithTimeout(context.Background(), 3000000000)
	defer cancel()
	elements, _ := finder.ManagedObjectList(ctx, folderPath)

	permissions, err := m.RetrieveEntityPermissions(ctx, elements[0].Object.Reference(), true)
	if err != nil {
		return nil, err
	}

	for _, permission := range permissions {
		if permission.Principal == principal {
			return &permission, nil
		}
	}

	return nil, errors.New("There is no prinicipal with name " + principal)
}

// Create Entity Permission
func Create(client *govmomi.Client, principal string, folderPath string, roleID int, group bool, propagate bool) error {
	m := object.NewAuthorizationManager(client.Client)
	finder := find.NewFinder(client.Client, true)
	ctx, cancel := context.WithTimeout(context.Background(), 3000000000)
	defer cancel()
	elements, err := finder.ManagedObjectList(ctx, folderPath)

	if err != nil {
		return err
	}

	if len(elements) == 0 {
		return errors.New("No Entity found inside " + folderPath)
	}

	perms := []types.Permission{types.Permission{
		Principal: principal,
		RoleId:    int32(roleID),
		Group:     group,
		Propagate: propagate,
	}}

	return m.SetEntityPermissions(ctx, elements[0].Object.Reference(), perms)
}

// Remove Entity Permission
func Remove(client *govmomi.Client, permission *types.Permission) error {
	m := object.NewAuthorizationManager(client.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 3000000000)
	defer cancel()

	return m.RemoveEntityPermission(ctx, permission.Entity.Reference(), permission.Principal, permission.Group)
}
