# GraphML library for Go

The goal of the project is to provide a lossless encoder and decoder for
GraphML.

## Status

The library can successfully roundtrip GraphML of following applications:

- [yEd](https://www.yworks.com/products/yed) 3.18

- [Gephi](https://gephi.org/) 0.9

- [Cytoscape](http://www.cytoscape.org/) 3.6

**Supported features:**

- Nodes, Edges

- Multiple graphs

- Subgraphs inside nodes

- Custom data for any node kind (`key`, `data`)

**Not yet supported:**

- Hyper-Edges

- Ports

- Endpoints