package graphml

import (
	"encoding/xml"
	"io"
)

const namespace = "http://graphml.graphdrawing.org/xmlns"

type element interface {
	addAttr(a xml.Attr)
	attrs() []xml.Attr
}

func newAttr(ns, local, value string) xml.Attr {
	return xml.Attr{Name: xml.Name{Local: local, Space: ns}, Value: value}
}

type Document struct {
	Instr  xml.ProcInst
	Attrs  []xml.Attr
	Keys   []Key
	Graphs []Graph `xml:"graph"`
	Data   []Data  `xml:"data"`
}

type Object struct {
	ID           string     `xml:"id,attr"`
	Unrecognized []xml.Attr `xml:",any,attr"`
}

func (o *Object) addAttr(a xml.Attr) {
	switch a.Name.Local {
	case "id":
		o.ID = a.Value
	default:
		o.Unrecognized = append(o.Unrecognized, a)
	}
}
func (o *Object) attrs() []xml.Attr {
	out := make([]xml.Attr, len(o.Unrecognized)+1)
	out = append(out, newAttr("", "id", o.ID))
	out = append(out, o.Unrecognized...)
	return out
}

type ExtObject struct {
	Object
	Data []Data `xml:"data"`
}

func NewKey(kind Kind, id, name, typ string) Key {
	return Key{
		Object: Object{
			ID: id,
		},
		For:  kind,
		Name: name, Type: typ,
	}
}

type Key struct {
	Object
	For  Kind   `xml:"for,attr"`
	Name string `xml:"attr.name,attr"`
	Type string `xml:"attr.type,attr"`
}

func (k *Key) addAttr(a xml.Attr) {
	switch a.Name.Local {
	case "for":
		k.For = Kind(a.Value)
	case "attr.name":
		k.Name = a.Value
	case "attr.type":
		k.Type = a.Value
	default:
		k.Object.addAttr(a)
	}
}
func (k *Key) attrs() []xml.Attr {
	attrs := k.Object.attrs()
	attrs = append(attrs, newAttr("", "for", string(k.For)))
	if k.Name != "" {
		attrs = append(attrs, newAttr("", "attr.name", k.Name))
	}
	if k.Type != "" {
		attrs = append(attrs, newAttr("", "attr.type", k.Type))
	}
	return attrs
}

type Graph struct {
	ExtObject
	EdgeDefault EdgeDir `xml:"edgedefault,attr"`

	Nodes []Node `xml:"node"`
	Edges []Edge `xml:"edge"`
}

func (g *Graph) addAttr(a xml.Attr) {
	switch a.Name.Local {
	case "edgedefault":
		g.EdgeDefault = EdgeDir(a.Value)
	default:
		g.Object.addAttr(a)
	}
}
func (g *Graph) attrs() []xml.Attr {
	attrs := g.Object.attrs()
	if g.EdgeDefault != "" {
		attrs = append(attrs, newAttr("", "edgedefault", string(g.EdgeDefault)))
	}
	return attrs
}

type Node struct {
	ExtObject

	Graphs []Graph `xml:"graph"`
}

func (n *Node) addAttr(a xml.Attr) {
	n.Object.addAttr(a)
}
func (n *Node) attrs() []xml.Attr {
	return n.Object.attrs()
}

type Edge struct {
	ExtObject
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
}

func (e *Edge) addAttr(a xml.Attr) {
	switch a.Name.Local {
	case "source":
		e.Source = a.Value
	case "target":
		e.Target = a.Value
	default:
		e.Object.addAttr(a)
	}
}
func (e *Edge) attrs() []xml.Attr {
	attrs := e.Object.attrs()
	attrs = append(attrs,
		newAttr("", "source", e.Source),
		newAttr("", "target", e.Target),
	)
	return attrs
}

type Data struct {
	Key          string     `xml:"key,attr"`
	Unrecognized []xml.Attr `xml:",any,attr"`
	Data         []xml.Token
}

func (d *Data) Reader() xml.TokenReader {
	return &tokenReader{tokens: d.Data}
}
func (d *Data) addAttr(a xml.Attr) {
	switch a.Name.Local {
	case "key":
		d.Key = a.Value
	default:
		d.Unrecognized = append(d.Unrecognized, a)
	}
}
func (d *Data) attrs() []xml.Attr {
	attrs := make([]xml.Attr, len(d.Unrecognized)+1)
	attrs = append(attrs, newAttr("", "key", d.Key))
	attrs = append(attrs, d.Unrecognized...)
	return attrs
}

type tokenReader struct {
	tokens []xml.Token
}

func (r *tokenReader) Token() (xml.Token, error) {
	if len(r.tokens) == 0 {
		return nil, io.EOF
	}
	t := r.tokens[0]
	r.tokens = r.tokens[1:]
	return t, nil
}

type EdgeDir string

const (
	EdgeDirected   = EdgeDir("directed")
	EdgeUndirected = EdgeDir("undirected")
)

type Kind string

const (
	KindUnknown   = Kind("")
	KindAll       = Kind("all")
	KindGraphML   = Kind("graphml")
	KindGraph     = Kind("graph")
	KindNode      = Kind("node")
	KindEdge      = Kind("edge")
	KindHyperEdge = Kind("hyperedge")
	KindPort      = Kind("port")
	KindEndpoint  = Kind("endpoint")
)
