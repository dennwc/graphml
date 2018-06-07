package graphml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

func Decode(r io.Reader) (*Document, error) {
	dec := xml.NewDecoder(r)
	return DecodeFrom(dec)
}

func DecodeFrom(dec *xml.Decoder) (*Document, error) {
	b := &docDecoder{
		doc:  new(Document),
		keys: make(map[string]Key),
		ids:  make(map[string]struct{}),
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

type docDecoder struct {
	dec  *xml.Decoder
	keys map[string]Key
	ids  map[string]struct{}

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
			if t.Name.Local == "graphml" && t.Name.Space == namespace {
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
			if t.Name.Space != namespace {
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
	if _, ok := d.keys[k.ID]; ok {
		return fmt.Errorf("redefinition of key %q", k.ID)
	}
	d.keys[k.ID] = k
	d.doc.Keys = append(d.doc.Keys, k)
	if err := d.expectEnd(start.Name); err != nil {
		return err
	}
	return nil
}
func (d *docDecoder) decodeGraph(start xml.StartElement) (*Graph, error) {
	var g Graph
	for _, a := range start.Attr {
		g.addAttr(a)
	}
	if _, ok := d.ids[g.ID]; ok {
		return nil, fmt.Errorf("redefinition of id %q", g.ID)
	}
	d.ids[g.ID] = struct{}{}
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
			if t.Name.Space != namespace {
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
	if k, ok := d.keys[data.Key]; !ok || k.For != kind {
		return nil, fmt.Errorf("unexpected attr for %v: %q", kind, data.Key)
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
	if _, ok := d.ids[n.ID]; ok {
		return nil, fmt.Errorf("redefinition of id %q", n.ID)
	}
	d.ids[n.ID] = struct{}{}
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
			if t.Name.Space != namespace {
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
	if _, ok := d.ids[e.ID]; ok {
		return nil, fmt.Errorf("redefinition of id %q", e.ID)
	}
	d.ids[e.ID] = struct{}{}
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
			if t.Name.Space != namespace {
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
