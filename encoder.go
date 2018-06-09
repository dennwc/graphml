package graphml

import (
	"encoding/xml"
	"io"
)

// Encode writes a GraphML document to the stream.
func Encode(w io.Writer, doc *Document) error {
	enc := xml.NewEncoder(w)
	return EncodeTo(enc, doc)
}

// EncodeTo is similar to Encode, but allows to provide a custom XML encoder.
func EncodeTo(enc *xml.Encoder, doc *Document) error {
	d := &docEncoder{enc: enc}
	if err := d.Encode(doc); err != nil {
		return err
	}
	return enc.Flush()
}

func mlName(name string) xml.Name {
	return xml.Name{Local: name}
}

type docEncoder struct {
	enc *xml.Encoder
	err error
}

func (d *docEncoder) token(t xml.Token) error {
	if d.err == nil {
		d.err = d.enc.EncodeToken(t)
	}
	return d.err
}
func (d *docEncoder) start(name xml.Name, attrs []xml.Attr) error {
	return d.token(xml.StartElement{Name: name, Attr: attrs})
}
func (d *docEncoder) end(name xml.Name) error {
	return d.token(xml.EndElement{Name: name})
}
func (d *docEncoder) startEnd(name xml.Name, attrs []xml.Attr) error {
	t := xml.StartElement{Name: name, Attr: attrs}
	if err := d.token(t); err != nil {
		return err
	}
	return d.token(t.End())
}
func (d *docEncoder) Encode(doc *Document) error {
	if err := d.token(doc.Instr); err != nil {
		return err
	}
	if err := d.start(mlName("graphml"), doc.Attrs); err != nil {
		return err
	}
	for _, k := range doc.Keys {
		if err := d.startEnd(mlName("key"), k.attrs()); err != nil {
			return err
		}
	}
	for _, g := range doc.Graphs {
		if err := d.encodeGraph(&g); err != nil {
			return err
		}
	}
	if err := d.encodeData(doc.Data); err != nil {
		return err
	}
	return d.end(mlName("graphml"))
}
func (d *docEncoder) encodeData(data []Data) error {
	for _, dt := range data {
		if err := d.start(mlName("data"), dt.attrs()); err != nil {
			return err
		}
		for _, t := range dt.Data {
			if err := d.token(t); err != nil {
				return err
			}
		}
		if err := d.end(mlName("data")); err != nil {
			return err
		}
	}
	return nil
}
func (d *docEncoder) encodeGraph(g *Graph) error {
	if err := d.start(mlName("graph"), g.attrs()); err != nil {
		return err
	}
	if err := d.encodeData(g.Data); err != nil {
		return err
	}
	for _, n := range g.Nodes {
		if err := d.encodeNode(&n); err != nil {
			return err
		}
	}
	for _, e := range g.Edges {
		if err := d.encodeEdge(&e); err != nil {
			return err
		}
	}
	return d.end(mlName("graph"))
}
func (d *docEncoder) encodeNode(n *Node) error {
	if err := d.start(mlName("node"), n.attrs()); err != nil {
		return err
	}
	if err := d.encodeData(n.Data); err != nil {
		return err
	}
	for _, g := range n.Graphs {
		if err := d.encodeGraph(&g); err != nil {
			return err
		}
	}
	return d.end(mlName("node"))
}
func (d *docEncoder) encodeEdge(e *Edge) error {
	if err := d.start(mlName("edge"), e.attrs()); err != nil {
		return err
	}
	if err := d.encodeData(e.Data); err != nil {
		return err
	}
	return d.end(mlName("edge"))
}
