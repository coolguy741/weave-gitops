import Slider from "@material-ui/core/Slider";
import * as d3 from "d3";
import dagreD3 from "dagre-d3";
import _ from "lodash";
import * as React from "react";
import styled from "styled-components";
import { muiTheme } from "../lib/theme";
import Flex from "./Flex";
import Spacer from "./Spacer";
import Text from "./Text";

const loadingText = `Fetching all reconciled objects, \
    building directional relationship, \
    determining central node...`;

const SliderFlex = styled(Flex)`
  padding-top: ${(props) => props.theme.spacing.base};
  min-height: 400px;
  min-width: 60px;
  height: 100%;
  width: 5%;
`;

const PercentFlex = styled(Flex)`
  color: ${muiTheme.palette.primary.main};
  padding: 10px;
  background: rgba(0, 179, 236, 0.1);
  border-radius: 2px;
`;

const GraphFlex = styled(Flex)`
  position: relative;
`;

const LoadingText = styled(Text)`
  position: absolute;
`;

const Svg = styled.svg`
  &.loading {
    opacity: 0;
    pointer-events: none;
  }
`;

function calculateZoomRatio(zoomPercent): number {
  return (zoomPercent + 20) / 1500;
}

type DirectedGraphState = {
  zoomRatio: number;
  nodeOffsetX: number;
};

type LabelType = "html" | "text";
type LabelShape = "rect" | "ellipse";

type Props<N> = {
  className?: string;
  nodes: { id: any; data: N; label: (v: N) => string }[];
  edges: { source: any; target: any }[];
  scale: number;
  labelType?: LabelType;
  labelShape?: LabelShape;
};

function DirectedGraph<T>({
  className,
  nodes,
  edges,
  scale,
  labelType,
  labelShape,
}: Props<T>) {
  const svgRef = React.useRef();
  const graphRef = React.useRef<D3Graph>();

  const [zoomPercent, setZoomPercent] = React.useState<number>(scale);
  const [state, setState] = React.useState<DirectedGraphState>({
    zoomRatio: calculateZoomRatio(scale),
    nodeOffsetX: 0,
  });

  React.useEffect(() => {
    if (!svgRef.current) {
      return;
    }

    // https://github.com/jsdom/jsdom/issues/2531
    if (process.env.NODE_ENV === "test") {
      return;
    }

    const { nodeOffsetX } = state;

    const graph = new D3Graph(svgRef.current, {
      labelShape,
      labelType,
      initialZoom: scale,
      initialNodeOffsetX: nodeOffsetX,
    });
    graph.update(nodes, edges);
    graph.render();
    graphRef.current = graph;
  }, []);

  React.useEffect(() => {
    const { nodeOffsetX } = state;

    let newNodeOffsetX = 0;

    if (nodeOffsetX === 0) {
      const d3Graph = graphRef.current.graph;
      const graphNodes = d3Graph ? d3Graph.nodes() : null;
      const rootNode = graphNodes
        ? d3Graph.node(graphNodes[graphNodes.length - 1])
        : null;

      const newZoomRatio = calculateZoomRatio(zoomPercent);
      newNodeOffsetX = rootNode
        ? -zoomPercent * 1.25 + (rootNode.x - rootNode.width) * newZoomRatio
        : 0;

      setState({
        nodeOffsetX: newNodeOffsetX,
        zoomRatio: newZoomRatio,
      });
    }

    graphRef.current.zoom(zoomPercent, nodeOffsetX || newNodeOffsetX);
  }, [zoomPercent]);

  React.useEffect(() => {
    graphRef.current.update(nodes, edges);
    graphRef.current.render();
  }, [nodes, edges]);

  const { nodeOffsetX } = state;

  const isLoadingGraph = nodeOffsetX === 0;

  return (
    <GraphFlex wide tall className={className}>
      {isLoadingGraph && <LoadingText>{loadingText}</LoadingText>}

      <Svg
        viewBox="0 0 100 100"
        preserveAspectRatio="xMidYMid meet"
        ref={svgRef}
        className={isLoadingGraph ? "loading" : ""}
      />

      <Flex tall>
        <SliderFlex column center align>
          <Slider
            onChange={(e, value: number) => setZoomPercent(value)}
            defaultValue={20}
            orientation="vertical"
            aria-label="zoom"
          />
          <Spacer padding="base" />
          <PercentFlex>{zoomPercent}%</PercentFlex>
        </SliderFlex>
      </Flex>
    </GraphFlex>
  );
}

export default styled(DirectedGraph)`
  overflow: hidden;
  text {
    font-weight: 300;
    font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
    font-size: 12px;
  }
  .edgePath path {
    stroke: #333;
    stroke-width: 1.5px;
  }
  foreignObject {
    display: flex;
    flex-direction: column;
    width: 650px;
    height: 200px;
    overflow: visible;
  }
  .MuiSlider-vertical {
    min-height: 400px;
  }
  .MuiSlider-vertical .MuiSlider-track {
    width: 6px;
  }
  .MuiSlider-vertical .MuiSlider-rail {
    width: 6px;
  }
  .MuiSlider-vertical .MuiSlider-thumb {
    margin-left: -9px;
  }
  .MuiSlider-thumb {
    width: 24px;
    height: 24px;
  }
`;

type D3GraphOptions = {
  initialZoom: number;
  initialNodeOffsetX: number;
  labelType?: LabelType;
  labelShape?: LabelShape;
};

// D3 doesn't play nicely with React, as they both manipulate the DOM.
// Since polling was added, we re-render the graph after every poll.
// Split the D3 graphic logic into a class to avoid resetting transform state on every render.
class D3Graph {
  containerEl;
  svg;
  graph;
  opts;

  constructor(element, opts: D3GraphOptions) {
    const dagreD3LibRef = dagreD3;
    this.graph = new dagreD3LibRef.graphlib.Graph();
    this.opts = opts;
    this.containerEl = element;
    this.svg = d3.select(element);
    this.svg.append("g");
    this.zoom(opts.initialZoom, opts.initialNodeOffsetX);
  }

  zoom(zoomPercent, nodeOffsetX) {
    const zoom = d3.zoom().on("zoom", (e) => {
      e.transform.k = (zoomPercent + 20) / 1500;
      this.svg.select("g").attr("transform", e.transform);
    });

    this.svg
      .call(zoom)
      .call(
        zoom.transform,
        d3.zoomIdentity.translate(-nodeOffsetX, 0).scale(zoomPercent)
      )
      .on("wheel.zoom", null);
  }

  update(nodes, edges) {
    this.graph
      .setGraph({
        nodesep: 100,
        ranksep: 150,
        rankdir: "TB",
      })
      .setDefaultEdgeLabel(() => {
        return {};
      });

    _.each(nodes, (n) => {
      this.graph.setNode(n.id, {
        label: n.label(n.data),
        labelType: this.opts.labelType,
        shape: this.opts.labelShape,
        width: 630,
        height: 180,
        rx: 10,
        ry: 10,
        labelClass: "node-label",
      });
    });

    _.each(edges, (e) => {
      this.graph.setEdge(e.source, e.target, { arrowhead: "undirected" });
    });
  }

  render() {
    const render = new dagreD3.render();
    render(d3.select("svg g"), this.graph);
  }
}
