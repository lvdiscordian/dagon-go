package dagon

type Middleware func(Handler) Handler

type MiddlewareChain struct {
	middleware []Middleware
}

func NewMiddlewareChain(middleware ...Middleware) MiddlewareChain {
	return MiddlewareChain{append(([]Middleware)(nil), middleware...)}
}

func (mc MiddlewareChain) Finally(h Handler) Handler {
	for i := len(mc.middleware) - 1; i >= 0; i-- {
		h = mc.middleware[i](h)
	}
	return h
}

func (mc MiddlewareChain) Append(middleware ...Middleware) MiddlewareChain {
	newChain := make([]Middleware, len(mc.middleware)+len(middleware))
	copy(newChain, mc.middleware)
	copy(newChain[len(mc.middleware):], middleware)
	return NewMiddlewareChain(newChain...)
}

func (mc MiddlewareChain) Extend(chain MiddlewareChain) MiddlewareChain {
	return mc.Append(chain.middleware...)
}
