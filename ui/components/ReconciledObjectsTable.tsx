import _ from "lodash";
import * as React from "react";
import styled from "styled-components";
import { useGetReconciledObjects } from "../hooks/flux";
import {
  FluxObjectKind,
  GroupVersionKind,
  UnstructuredObject,
} from "../lib/api/core/types.pb";
import { formatURL, sourceTypeToRoute } from "../lib/nav";
import { NoNamespace } from "../lib/types";
import { statusSortHelper } from "../lib/utils";
import { SortType } from "./DataTable";
import FilterableTable, {
  filterConfigForStatus,
  filterConfigForString,
} from "./FilterableTable";
import KubeStatusIndicator, { computeMessage } from "./KubeStatusIndicator";
import Link from "./Link";
import RequestStateHandler from "./RequestStateHandler";

export interface ReconciledVisualizationProps {
  className?: string;
  automationName: string;
  namespace?: string;
  automationKind: FluxObjectKind;
  kinds: GroupVersionKind[];
  clusterName: string;
}

function ReconciledObjectsTable({
  className,
  automationName,
  namespace = NoNamespace,
  automationKind,
  kinds,
  clusterName,
}: ReconciledVisualizationProps) {
  const {
    data: objs,
    error,
    isLoading,
  } = useGetReconciledObjects(
    automationName,
    namespace,
    automationKind,
    kinds,
    clusterName
  );

  const initialFilterState = {
    ...filterConfigForString(objs, "namespace"),
    ...filterConfigForStatus(objs),
  };

  const kindsToAddLinksFrom = [
    FluxObjectKind.KindKustomization,
    FluxObjectKind.KindHelmRelease,
  ];

  const shouldAddLinks = kindsToAddLinksFrom.includes(automationKind);

  const kindsToAddLinksTo = [
    FluxObjectKind.KindKustomization,
    FluxObjectKind.KindHelmRelease,
    FluxObjectKind.KindGitRepository,
    FluxObjectKind.KindHelmRepository,
    FluxObjectKind.KindBucket,
  ];

  return (
    <RequestStateHandler loading={isLoading} error={error}>
      <FilterableTable
        filters={initialFilterState}
        className={className}
        fields={[
          {
            value: (u: UnstructuredObject) => {
              const kind = FluxObjectKind[`Kind${u.groupVersionKind.kind}`];

              return shouldAddLinks &&
                kind &&
                kindsToAddLinksTo.includes(kind) ? (
                <Link
                  to={formatURL(sourceTypeToRoute(kind), {
                    name: u.name,
                    namespace: u.namespace,
                    clusterName: u.clusterName,
                  })}
                >
                  {u.name}
                </Link>
              ) : (
                "name"
              );
            },
            label: "Name",
            maxWidth: 600,
          },
          {
            label: "Type",
            value: (u: UnstructuredObject) => u.groupVersionKind.kind,
            sortType: SortType.string,
            sortValue: (u: UnstructuredObject) => u.groupVersionKind.kind,
          },
          {
            label: "Namespace",
            value: "namespace",
            sortType: SortType.string,
            sortValue: ({ namespace }) => namespace,
          },
          {
            label: "Status",
            value: (u: UnstructuredObject) =>
              u.conditions.length > 0 ? (
                <KubeStatusIndicator
                  conditions={u.conditions}
                  suspended={u.suspended}
                  short
                />
              ) : null,
            sortType: SortType.number,
            sortValue: statusSortHelper,
          },
          {
            label: "Message",
            value: (u: UnstructuredObject) => _.first(u.conditions)?.message,
            sortType: SortType.string,
            sortValue: ({ conditions }) => computeMessage(conditions),
            maxWidth: 600,
          },
        ]}
        rows={objs}
      />
    </RequestStateHandler>
  );
}

export default styled(ReconciledObjectsTable).attrs({
  className: ReconciledObjectsTable.name,
})`
  td:nth-child(5) {
    white-space: pre-wrap;
    overflow-wrap: break-word;
    word-wrap: break-word;
  }
`;
