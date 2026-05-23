package memory

const (
	RED   = true
	BLACK = false
)

type Memtable struct {
	root *Node
	size int
}

type Node struct {
	key        string
	data       []byte
	color      bool
	leftchild  *Node
	rightchild *Node
	parent     *Node
}

// shared sentinel leaf — always BLACK, never nil
var nilNode = &Node{color: BLACK}

func NewMemtable() Memtable {
	return Memtable{root: nilNode}
}

func newNode(key string, data []byte) *Node {
	return &Node{
		key:        key,
		data:       data,
		color:      RED,
		leftchild:  nilNode,
		rightchild: nilNode,
		parent:     nilNode,
	}
}

// ── Public API ────────────────────────────────────────────────────────────────

func (mt *Memtable) Insert(key string, data []byte) {
	nd := newNode(key, data)

	parent := nilNode
	cur := mt.root
	for cur != nilNode {
		parent = cur
		switch {
		case key < cur.key:
			cur = cur.leftchild
		case key > cur.key:
			cur = cur.rightchild
		default:
			cur.data = data // update existing key
			return
		}
	}

	nd.parent = parent
	switch {
	case parent == nilNode:
		mt.root = nd
	case key < parent.key:
		parent.leftchild = nd
	default:
		parent.rightchild = nd
	}

	mt.size++
	mt.fixInsert(nd)
}

func (mt *Memtable) Search(key string) ([]byte, bool) {
	nd := mt.search(mt.root, key)
	if nd == nilNode {
		return nil, false
	}
	return nd.data, true
}

func (mt *Memtable) Delete(key string) bool {
	nd := mt.search(mt.root, key)
	if nd == nilNode {
		return false
	}
	mt.delete(nd)
	mt.size--
	return true
}

func (mt *Memtable) Size() int { return mt.size }

// InOrder returns all entries sorted by key — used when flushing to SSTable.
func (mt *Memtable) InOrder() []KV {
	out := make([]KV, 0, mt.size)
	mt.inorder(mt.root, &out)
	return out
}

type KV struct {
	Key  string
	Data []byte
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (mt *Memtable) search(nd *Node, key string) *Node {
	for nd != nilNode {
		switch {
		case key < nd.key:
			nd = nd.leftchild
		case key > nd.key:
			nd = nd.rightchild
		default:
			return nd
		}
	}
	return nilNode
}

func (mt *Memtable) inorder(nd *Node, out *[]KV) {
	if nd == nilNode {
		return
	}
	mt.inorder(nd.leftchild, out)
	*out = append(*out, KV{nd.key, nd.data})
	mt.inorder(nd.rightchild, out)
}

// ── Rotations ─────────────────────────────────────────────────────────────────

func (mt *Memtable) rotateLeft(x *Node) {
	y := x.rightchild
	x.rightchild = y.leftchild
	if y.leftchild != nilNode {
		y.leftchild.parent = x
	}
	y.parent = x.parent
	switch {
	case x.parent == nilNode:
		mt.root = y
	case x == x.parent.leftchild:
		x.parent.leftchild = y
	default:
		x.parent.rightchild = y
	}
	y.leftchild = x
	x.parent = y
}

func (mt *Memtable) rotateRight(x *Node) {
	y := x.leftchild
	x.leftchild = y.rightchild
	if y.rightchild != nilNode {
		y.rightchild.parent = x
	}
	y.parent = x.parent
	switch {
	case x.parent == nilNode:
		mt.root = y
	case x == x.parent.rightchild:
		x.parent.rightchild = y
	default:
		x.parent.leftchild = y
	}
	y.rightchild = x
	x.parent = y
}

// ── Insert fixup ──────────────────────────────────────────────────────────────

func (mt *Memtable) fixInsert(z *Node) {
	for z.parent.color == RED {
		if z.parent == z.parent.parent.leftchild {
			uncle := z.parent.parent.rightchild
			if uncle.color == RED {
				// Case 1: uncle RED → recolor, move z up
				z.parent.color = BLACK
				uncle.color = BLACK
				z.parent.parent.color = RED
				z = z.parent.parent
			} else {
				if z == z.parent.rightchild {
					// Case 2: z is inner child → rotate to outer
					z = z.parent
					mt.rotateLeft(z)
				}
				// Case 3: z is outer child → rotate + recolor
				z.parent.color = BLACK
				z.parent.parent.color = RED
				mt.rotateRight(z.parent.parent)
			}
		} else {
			// Mirror of above
			uncle := z.parent.parent.leftchild
			if uncle.color == RED {
				z.parent.color = BLACK
				uncle.color = BLACK
				z.parent.parent.color = RED
				z = z.parent.parent
			} else {
				if z == z.parent.leftchild {
					z = z.parent
					mt.rotateRight(z)
				}
				z.parent.color = BLACK
				z.parent.parent.color = RED
				mt.rotateLeft(z.parent.parent)
			}
		}
	}
	mt.root.color = BLACK
}

// ── Delete ────────────────────────────────────────────────────────────────────

func (mt *Memtable) delete(z *Node) {
	y := z
	yOrigColor := y.color
	var x *Node

	switch {
	case z.leftchild == nilNode:
		x = z.rightchild
		mt.transplant(z, z.rightchild)
	case z.rightchild == nilNode:
		x = z.leftchild
		mt.transplant(z, z.leftchild)
	default:
		y = minimum(z.rightchild) // in-order successor
		yOrigColor = y.color
		x = y.rightchild
		if y.parent == z {
			x.parent = y
		} else {
			mt.transplant(y, y.rightchild)
			y.rightchild = z.rightchild
			y.rightchild.parent = y
		}
		mt.transplant(z, y)
		y.leftchild = z.leftchild
		y.leftchild.parent = y
		y.color = z.color
	}

	if yOrigColor == BLACK {
		mt.fixDelete(x)
	}
}

func (mt *Memtable) transplant(u, v *Node) {
	switch {
	case u.parent == nilNode:
		mt.root = v
	case u == u.parent.leftchild:
		u.parent.leftchild = v
	default:
		u.parent.rightchild = v
	}
	v.parent = u.parent
}

func minimum(nd *Node) *Node {
	for nd.leftchild != nilNode {
		nd = nd.leftchild
	}
	return nd
}

// ── Delete fixup ──────────────────────────────────────────────────────────────

func (mt *Memtable) fixDelete(x *Node) {
	for x != mt.root && x.color == BLACK {
		if x == x.parent.leftchild {
			w := x.parent.rightchild
			if w.color == RED {
				// Case 1: sibling RED → rotate, convert to case 2/3/4
				w.color = BLACK
				x.parent.color = RED
				mt.rotateLeft(x.parent)
				w = x.parent.rightchild
			}
			if w.leftchild.color == BLACK && w.rightchild.color == BLACK {
				// Case 2: sibling's children both BLACK → push black up
				w.color = RED
				x = x.parent
			} else {
				if w.rightchild.color == BLACK {
					// Case 3: sibling's right child BLACK → rotate sibling
					w.leftchild.color = BLACK
					w.color = RED
					mt.rotateRight(w)
					w = x.parent.rightchild
				}
				// Case 4: sibling's right child RED → rotate parent
				w.color = x.parent.color
				x.parent.color = BLACK
				w.rightchild.color = BLACK
				mt.rotateLeft(x.parent)
				x = mt.root
			}
		} else {
			// Mirror
			w := x.parent.leftchild
			if w.color == RED {
				w.color = BLACK
				x.parent.color = RED
				mt.rotateRight(x.parent)
				w = x.parent.leftchild
			}
			if w.rightchild.color == BLACK && w.leftchild.color == BLACK {
				w.color = RED
				x = x.parent
			} else {
				if w.leftchild.color == BLACK {
					w.rightchild.color = BLACK
					w.color = RED
					mt.rotateLeft(w)
					w = x.parent.leftchild
				}
				w.color = x.parent.color
				x.parent.color = BLACK
				w.leftchild.color = BLACK
				mt.rotateRight(x.parent)
				x = mt.root
			}
		}
	}
	x.color = BLACK
}
