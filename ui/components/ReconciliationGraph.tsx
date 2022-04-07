import _ from "lodash";
import * as React from "react";
import { renderToString } from "react-dom/server";
import styled from "styled-components";
/*eslint import/no-unresolved: [0]*/
//@ts-ignore
import failedSrc from "url:../images/failed.svg";
//@ts-ignore
import successSrc from "url:../images/success.svg";
//@ts-ignore
import suspendedSrc from "url:../images/suspended.svg";
import { useGetReconciledObjects } from "../hooks/flux";
import { Condition, UnstructuredObject } from "../lib/api/core/types.pb";
import DirectedGraph from "./DirectedGraph";
import Flex from "./Flex";
import { computeReady } from "./KubeStatusIndicator";
import { ReconciledVisualizationProps } from "./ReconciledObjectsTable";
import RequestStateHandler from "./RequestStateHandler";

export type Props = ReconciledVisualizationProps & {
  parentObject: {
    name?: string;
    namespace?: string;
    conditions?: Condition[];
    suspended?: boolean;
  };
};

const GraphIcon = styled.img`
  height: 32px;
  width: 32px;
`;

function getStatusIcon(status: string, suspended: boolean) {
  if (suspended) return <GraphIcon src={suspendedSrc} />;
  switch (status) {
    case "Current":
      return <GraphIcon src={successSrc} />;

    case "InProgress":
      return <GraphIcon src={suspendedSrc} />;

    case "Failed":
      return <GraphIcon src={failedSrc} />;

    default:
      return "";
  }
}

type NodeHtmlProps = {
  object: UnstructuredObject;
};

const NodeHtml = ({ object }: NodeHtmlProps) => {
  return (
    <div className="node">
      <Flex
        className={`status-line ${
          object.suspended ? "InProgress" : object.status
        }`}
      />
      <Flex column className="nodeText">
        <Flex start wide align className="name">
          <div
            className={`status ${
              object.suspended ? "InProgress" : object.status
            }`}
          >
            {getStatusIcon(object.status, object.suspended)}
          </div>
          <div style={{ padding: 4 }} />
          {object.name}
        </Flex>
        <Flex start wide align className="kind">
          <div className="kind-text">{object.groupVersionKind.kind}</div>
        </Flex>
        <Flex start wide align className="kind">
          <div className="kind-text">{object.namespace}</div>
        </Flex>
      </Flex>
    </div>
  );
};

function ReconciliationGraph({
  className,
  parentObject,
  automationKind,
  kinds,
  clusterName,
}: Props) {
  const {
    data: objects,
    error,
    isLoading,
  } = parentObject ? useGetReconciledObjects(
    parentObject?.name,
    parentObject?.namespace,
    automationKind,
    kinds,
    clusterName
  ) : { data: [], error: null, isLoading: false };

  const edges = _.reduce(
    objects,
    (r, v: any) => {
      if (v.parentUid) {
        r.push({ source: v.parentUid, target: v.uid });
      } else {
        r.push({ source: parentObject.name, target: v.uid });
      }
      return r;
    },
    []
  );

  const findParentStatus = (parent) => {
    if (parent.suspended) return "InProgress";
    if (computeReady(parent.conditions)) return "Current";
    return "Failed";
  };

  const nodes = [
    ..._.map(objects, (r) => ({
      id: r.uid,
      data: r,
      label: (u: UnstructuredObject) => renderToString(<NodeHtml object={u} />),
    })),
    {
      id: parentObject.name,
      data: { ...parentObject, status: findParentStatus(parentObject) },
      label: (u: Props["parentObject"]) =>
        renderToString(
          <NodeHtml
            object={{ ...u, groupVersionKind: { kind: automationKind } }}
          />
        ),
    },
  ];

  return (
    <RequestStateHandler loading={isLoading} error={error}>
      <div className={className} style={{ height: "100%", width: "100%" }}>
        <DirectedGraph
          width="100%"
          height="100%"
          scale={1}
          nodes={nodes}
          edges={edges}
          labelShape="rect"
          labelType="html"
        />
      </div>
    </RequestStateHandler>
  );
}

export default styled(ReconciliationGraph)`
  .node {
    font-size: 16px;
    width: 650px;
    height: 200px;
    display: flex;
    justify-content: space-between;
  }

  rect {
    fill: white;
    stroke: ${(props) => props.theme.colors.neutral20};
    stroke-width: 3;
  }
  .status {
    display: flex;
    align-items: center;
  }
  .kind-text {
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 28px;
  }
  .status-line {
    width: 2.5%;
    border-radius: 10px 0px 0px 10px;
  }
  .nodeText {
    width: 95%;
    align-items: flex-start;
    justify-content: space-evenly;
  }

  .Current {
    color: ${(props) => props.theme.colors.success};
    &.status-line {
      background-color: ${(props) => props.theme.colors.success};
    }
  }
  .InProgress {
    color: ${(props) => props.theme.colors.suspended};
    &.status-line {
      background-color: ${(props) => props.theme.colors.suspended};
    }
  }
  .Failed {
    color: ${(props) => props.theme.colors.alert};
    &.status-line {
      background-color: ${(props) => props.theme.colors.alert};
    }
  }
  .name {
    color: ${(props) => props.theme.colors.black};
    font-weight: 800;
    font-size: 28px;
    white-space: pre-wrap;
  }
  .kind {
    color: ${(props) => props.theme.colors.neutral30};
  }
  .edgePath path {
    stroke: ${(props) => props.theme.colors.neutral30};
    stroke-width: 1px;
  }
`;
