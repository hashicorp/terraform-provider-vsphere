package sso

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-provider-vsphere/vsphere/internal/helper/provider"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/ssoadmin"
	"github.com/vmware/govmomi/ssoadmin/types"
	"github.com/vmware/govmomi/sts"
	"github.com/vmware/govmomi/vim25/soap"
)

func AdminPersonUserFromName(clientWrapper *SsoAdminClientConfig, name string) (*types.AdminPersonUser, error) {

	ctx, cancel := context.WithTimeout(context.Background(), provider.DefaultAPITimeout)
	defer cancel()

	c, err := newClientWithAuthentication(ctx, *clientWrapper)
	if err != nil {
		return nil, err
	}
	defer c.Logout(ctx)

	return c.FindPersonUser(ctx, name)
}

// Wrapper for lazy authenticated SSO admin requests
type SsoAdminClientConfig struct {
	VimClient *govmomi.Client
	UserInfo  *url.Userinfo
}

// Create a new SSO admin client with authentication on demand
// SSO admin server has its own session manager, so the govc persisted session cookies cannot
// be used to authenticate. We have to issue a new token each time.
func newClientWithAuthentication(ctx context.Context, s SsoAdminClientConfig) (*ssoadmin.Client, error) {

	vc := s.VimClient.Client

	c, err := ssoadmin.NewClient(ctx, vc)
	if err != nil {
		return nil, err
	}

	tokens, err := sts.NewClient(ctx, s.VimClient.Client)
	if err != nil {
		return nil, err
	}

	req := sts.TokenRequest{
		Certificate: vc.Certificate(),
		Userinfo:    s.UserInfo,
	}

	signer, err := tokens.Issue(ctx, req)
	if err != nil {
		return nil, err
	}

	if err = c.Login(c.WithHeader(ctx, soap.Header{Security: signer})); err != nil {
		return nil, err
	}

	return c, nil
}
