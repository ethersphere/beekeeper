package bee

import "context"

func (b *BeeV2) RefreshAuthToken(ctx context.Context, token string) (string, error) {
	return b.client.Refresh(ctx, token)
}

func (b *BeeV2) Authenticate(ctx context.Context, password string) (string, error) {
	return b.client.Authenticate(ctx, b.opts.Role, password)
}
