package graphml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// Decode reads a GraphML document from the stream.
func Decode(r io.Reader) (*Document, error) {
	dec := xml.NewDecoder(r)
	return DecodeFrom(dec)
}

// DecodeFrom is similar to Decode, but allows to specify a custom XML decoder.
func DecodeFrom(dec *xml.Decoder) (*Document, error) {
	b := &docDecoder{
		doc:     new(Document),
		keysAll: make(map[string]Key),
		keys:    make(map[docKey]Key),
		ids:     make(map[string]struct{}),
	}
	if err := b.DecodeFrom(dec); err != nil {
		return nil, err
	}
	return b.doc, nil
}

func canSkip(t xml.Token) bool {
	switch t := t.(type) {
	case xml.Comment:
		return true
	case xml.CharData:
		if len(bytes.TrimSpace([]byte(t))) == 0 {
			return true
		}
	}
	return false
}

type docKey struct {
	name string
	kind Kind
}

type docDecoder struct {
	dec     *xml.Decoder
	keysAll map[string]Key
	keys    map[docKey]Key
	ids     map[string]struct{}
	lastID  int

	doc *Document
}

func (d *docDecoder) token() (xml.Token, error) {
	return d.dec.Token()
}
func (d *docDecoder) expectEnd(tok xml.Name) error {
	for {
		t, err := d.token()
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		} else if err != nil {
			return err
		} else if canSkip(t) {
			continue
		}
		switch t := t.(type) {
		case xml.EndElement:
			if t.Name == tok {
				return nil
			}
		}
		return fmt.Errorf("unexpected token: %T: %#v", t, t)
	}
}
func (d *docDecoder) startGraphML() (xml.StartElement, error) {
	for {
		t, err := d.token()
		if err == io.EOF {
			return xml.StartElement{}, io.ErrUnexpectedEOF
		} else if err != nil {
			return xml.StartElement{}, err
		} else if canSkip(t) {
			continue
		}
		switch t := t.(type) {
		case xml.ProcInst:
			d.doc.Instr = t.Copy()
			continue
		case xml.StartElement:
			if t.Name.Local == "graphml" && t.Name.Space == Namespace {
				d.doc.Attrs = t.Copy().Attr
				return t, nil
			}
		}
		return xml.StartElement{}, fmt.Errorf("unexpected token: %T: %#v", t, t)
	}
}
func (d *docDecoder) DecodeFrom(dec *xml.Decoder) error {
	d.dec = dec
	start, err := d.startGraphML()
	if err != nil {
		return err
	}
	for {
		t, err := d.token()
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		} else if err != nil {
			return err
		} else if canSkip(t) {
			continue
		}
		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Space != Namespace {
				return fmt.Errorf("unexpected element: %v", t.Name)
			}
			switch t.Name.Local {
			case "key":
				if err := d.decodeKey(t); err != nil {
					return err
				}
			case "graph":
				g, err := d.decodeGraph(t)
				if err != nil {
					return err
				}
				d.doc.Graphs = append(d.doc.Graphs, *g)
			case "data":
				data, err := d.decodeData(KindGraphML, t)
				if err != nil {
					return err
				}
				d.doc.Data = append(d.doc.Data, *data)
			default:
				return fmt.Errorf("unknown element: %v", t.Name)
			}
			continue
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
		return fmt.Errorf("unexpected token: %T: %#v", t, t)
	}
}
func (d *docDecoder) decodeKey(start xml.StartElement) error {
	var k Key
	for _, a := range start.Attr {
		k.addAttr(a)
	}
	if k.For == "" {
		k.For = KindAll
	}
	if k.For == KindAll {
		if _, ok := d.keysAll[k.ID]; ok {
			return fmt.Errorf("redefinition of key %q", k.ID)
		}
		d.keysAll[k.ID] = k
	} else {
		dk := docKey{name: k.ID, kind: k.For}
		if _, ok := d.keys[dk]; ok {
			return fmt.Errorf("redefinition of key %q for %v", k.ID, k.For)
		}
		d.keys[dk] = k
	}
	d.doc.Keys = append(d.doc.Keys, k)
	if err := d.expectEnd(start.Name); err != nil {
		return err
	}
	return nil
}
func (d *docDecoder) addID(id string) (string, error) {
	if id == "" {
		return "", nil
	}
	if _, ok := d.ids[id]; ok {
		return "", fmt.Errorf("redefinition of id %q", id)
	}
	d.ids[id] = struct{}{}
	return id, nil
}
func (d *docDecoder) decodeGraph(start xml.StartElement) (*Graph, error) {
	var g Graph
	for _, a := range start.Attr {
		g.addAttr(a)
	}
	var err error
	g.ID, err = d.addID(g.ID)
	if err != nil {
		return nil, err
	}
	if err := d.decodeGraphNodes(&g, start); err != nil {
		return nil, err
	}
	return &g, nil
}
func (d *docDecoder) decodeGraphNodes(g *Graph, start xml.StartElement) error {
	for {
		t, err := d.token()
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		} else if err != nil {
			return err
		} else if canSkip(t) {
			continue
		}
		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Space != Namespace {
				return fmt.Errorf("unexpected element: %v", t.Name)
			}
			switch t.Name.Local {
			case "data":
				data, err := d.decodeData(KindGraph, t)
				if err != nil {
					return err
				}
				g.Data = append(g.Data, *data)
			case "node":
				n, err := d.decodeNode(t)
				if err != nil {
					return err
				}
				g.Nodes = append(g.Nodes, *n)
			case "edge":
				e, err := d.decodeEdge(t)
				if err != nil {
					return err
				}
				g.Edges = append(g.Edges, *e)
			default:
				return fmt.Errorf("unknown element: %v", t.Name)
			}
			continue
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
		return fmt.Errorf("unexpected token: %T: %#v", t, t)
	}
}
func (d *docDecoder) decodeData(kind Kind, start xml.StartElement) (*Data, error) {
	var data Data
	for _, a := range start.Attr {
		data.addAttr(a)
	}
	if _, ok := d.keys[docKey{name: data.Key, kind: kind}]; !ok {
		if _, ok = d.keysAll[data.Key]; !ok {
			return nil, fmt.Errorf("unexpected attr for %v: %q", kind, data.Key)
		}
	}
	for {
		t, err := d.token()
		if err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		} else if err != nil {
			return nil, err
		}
		switch e := t.(type) {
		case xml.EndElement:
			if e.Name == start.Name {
				return &data, nil
			}
		}
		t = xml.CopyToken(t)
		data.Data = append(data.Data, t)
	}
}
func (d *docDecoder) decodeNode(start xml.StartElement) (*Node, error) {
	var n Node
	for _, a := range start.Attr {
		n.addAttr(a)
	}
	var err error
	n.ID, err = d.addID(n.ID)
	if err != nil {
		return nil, err
	}
	for {
		t, err := d.token()
		if err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		} else if err != nil {
			return nil, err
		} else if canSkip(t) {
			continue
		}
		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Space != Namespace {
				return nil, fmt.Errorf("unexpected element: %v", t.Name)
			}
			switch t.Name.Local {
			case "data":
				data, err := d.decodeData(KindNode, t)
				if err != nil {
					return nil, err
				}
				n.Data = append(n.Data, *data)
			case "graph":
				g, err := d.decodeGraph(t)
				if err != nil {
					return nil, err
				}
				n.Graphs = append(n.Graphs, *g)
			default:
				return nil, fmt.Errorf("unknown element: %v", t.Name)
			}
			continue
		case xml.EndElement:
			if t.Name == start.Name {
				return &n, nil
			}
		}
		return nil, fmt.Errorf("unexpected token: %T: %#v", t, t)
	}
}
func (d *docDecoder) decodeEdge(start xml.StartElement) (*Edge, error) {
	var e Edge
	for _, a := range start.Attr {
		e.addAttr(a)
	}
	var err error
	e.ID, err = d.addID(e.ID)
	if err != nil {
		return nil, err
	}
	for {
		t, err := d.token()
		if err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		} else if err != nil {
			return nil, err
		} else if canSkip(t) {
			continue
		}
		switch t := t.(type) {
		case xml.StartElement:
			if t.Name.Space != Namespace {
				return nil, fmt.Errorf("unexpected element: %v", t.Name)
			}
			switch t.Name.Local {
			case "data":
				data, err := d.decodeData(KindEdge, t)
				if err != nil {
					return nil, err
				}
				e.Data = append(e.Data, *data)
			default:
				return nil, fmt.Errorf("unknown element: %v", t.Name)
			}
			continue
		case xml.EndElement:
			if t.Name == start.Name {
				return &e, nil
			}
		}
		return nil, fmt.Errorf("unexpected token: %T: %#v", t, t)
	}
}
