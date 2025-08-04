package router

type HandlerFunc func(params map[string]string)

type node struct {
	prefix      string
	children    []*node
	paramName   string
	isParam     bool
	isWildcard  bool // *
	isMultiWild bool // **
	handler     HandlerFunc
}

type Tree struct {
	staticRoutes map[string]HandlerFunc
	root         *node
}

type Router struct {
	trees map[string]*Tree // method → tree
}

func New() *Router {
	return &Router{
		trees: make(map[string]*Tree),
	}
}

func (r *Router) Register(method, path string, handler HandlerFunc) {
	if _, ok := r.trees[method]; !ok {
		r.trees[method] = &Tree{
			staticRoutes: make(map[string]HandlerFunc),
			root:         &node{},
		}
	}
	tree := r.trees[method]

	if isStaticPath(path) {
		tree.staticRoutes[path] = handler
		return
	}
	insert(tree.root, path, handler)
}

func (r *Router) Match(method, path string) (HandlerFunc, map[string]string, bool) {
	tree, ok := r.trees[method]
	if !ok {
		return nil, nil, false
	}
	if h, ok := tree.staticRoutes[path]; ok {
		return h, nil, true
	}
	params := map[string]string{}
	if h := match(tree.root, path, 0, params); h != nil {
		return h, params, true
	}
	return nil, nil, false
}

// ---- internals ----

func isStaticPath(path string) bool {
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '{', '*', '?':
			return false
		}
	}
	return true
}

func insert(n *node, path string, handler HandlerFunc) {
	i := 0
	for {
		if i >= len(path) {
			n.handler = handler
			return
		}
		start := i
		for i < len(path) && path[i] != '/' {
			i++
		}
		seg := path[start:i]
		if i < len(path) && path[i] == '/' {
			i++
		}
		child := findChild(n, seg)
		if child == nil {
			child = &node{prefix: seg}
			switch {
			case seg == "*":
				child.isWildcard = true
			case seg == "**":
				child.isMultiWild = true
			case isParam(seg):
				child.isParam = true
				child.paramName = seg[1 : len(seg)-1]
			}
			n.children = append(n.children, child)
		}
		if child.isParam {
			child.paramName = seg[1 : len(seg)-1]
		}
		n = child
	}
}

func findChild(n *node, seg string) *node {
	for _, ch := range n.children {
		if ch.prefix == seg {
			return ch
		}
		if ch.isParam && isParam(seg) && ch.paramName == seg[1:len(seg)-1] {
			return ch
		}
		if ch.isWildcard && seg == "*" {
			return ch
		}
		if ch.isMultiWild && seg == "**" {
			return ch
		}
	}
	return nil
}

func isParam(seg string) bool {
	return len(seg) > 1 && seg[0] == '{' && seg[len(seg)-1] == '}'
}

func match(n *node, path string, i int, params map[string]string) HandlerFunc {
	if i >= len(path) {
		if n.handler != nil {
			return n.handler
		}
		return nil
	}

	for _, ch := range n.children {
		pi := i
		newParams := copyParams(params) // 👈 param izolasyonu

		switch {
		case ch.isMultiWild:
			for j := len(path); j >= pi; j-- {
				if j < len(path) && path[j] != '/' {
					continue
				}
				if h := match(ch, path, j, newParams); h != nil {
					copyMap(params, newParams)
					return h
				}
			}

		case ch.isWildcard:
			end := pi
			for end < len(path) && path[end] != '/' {
				end++
			}
			if h := match(ch, path, end+1, newParams); h != nil {
				copyMap(params, newParams)
				return h
			}

		case ch.isParam:
			end := pi
			for end < len(path) && path[end] != '/' {
				end++
			}
			newParams[ch.paramName] = path[pi:end]
			if h := match(ch, path, end+1, newParams); h != nil {
				copyMap(params, newParams)
				return h
			}

		default:
			if len(path) >= pi+len(ch.prefix) && path[pi:pi+len(ch.prefix)] == ch.prefix {
				next := pi + len(ch.prefix)
				if next < len(path) && path[next] == '/' {
					next++
				}
				if h := match(ch, path, next, newParams); h != nil {
					copyMap(params, newParams)
					return h
				}
			}
		}
	}
	return nil
}

func copyParams(m map[string]string) map[string]string {
	cp := make(map[string]string, len(m)+1)
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func copyMap(dst, src map[string]string) {
	for k := range dst {
		delete(dst, k)
	}
	for k, v := range src {
		dst[k] = v
	}
}
