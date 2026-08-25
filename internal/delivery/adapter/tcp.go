package adapter

import (
	"context"
	"net"
)

type Dialer struct {
	net net.Dialer
	ctx context.Context
}

func (d *Dialer) DialContext(ctx context.Context, address string) (net.Conn, error) {
	if d.ctx == nil {
		d.ctx = ctx
	}
	return d.net.DialContext(d.ctx, "tcp", address)
}

func Dial(address string) (net.Conn, error) {
	return (&Dialer{}).DialContext(context.Background(), address)
}
