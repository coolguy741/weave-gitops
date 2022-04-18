import * as React from "react";
import styled from "styled-components";
import {
  Bucket,
  GitRepository,
  HelmChart,
  HelmRepository,
  SourceRefSourceKind,
} from "../lib/api/core/types.pb";
import { formatURL, sourceTypeToRoute } from "../lib/nav";
import { showInterval } from "../lib/time";
import { Source } from "../lib/types";
import { convertGitURLToGitProvider, statusSortHelper } from "../lib/utils";
import { SortType } from "./DataTable";
import FilterableTable, {
  filterConfigForStatus,
  filterConfigForString,
} from "./FilterableTable";
import KubeStatusIndicator, { computeMessage } from "./KubeStatusIndicator";
import Link from "./Link";
import Timestamp from "./Timestamp";

type Props = {
  className?: string;
  sources?: Source[];
  appName?: string;
};

function SourcesTable({ className, sources }: Props) {
  const [filterDialogOpen, setFilterDialog] = React.useState(false);

  const initialFilterState = {
    ...filterConfigForString(sources, "type"),
    ...filterConfigForStatus(sources),
    ...filterConfigForString(sources, "clusterName"),
  };

  return (
    <FilterableTable
      className={className}
      filters={initialFilterState}
      rows={sources}
      dialogOpen={filterDialogOpen}
      onDialogClose={() => setFilterDialog(false)}
      fields={[
        {
          label: "Name",
          value: (s: Source) => (
            <Link
              to={formatURL(sourceTypeToRoute(s.type), {
                name: s?.name,
                namespace: s?.namespace,
              })}
            >
              {s?.name}
            </Link>
          ),
          sortType: SortType.string,
          sortValue: (s: Source) => s.name || "",
          width: 5,
          textSearchable: true,
        },
        { label: "Type", value: "type", width: 5 },
        { label: "Namespace", value: "namespace", width: 5 },
        {
          label: "Status",
          value: (s: Source) => (
            <KubeStatusIndicator
              short
              conditions={s.conditions}
              suspended={s.suspended}
            />
          ),
          sortType: SortType.number,
          sortValue: statusSortHelper,
          width: 5,
        },
        {
          label: "Message",
          value: (s) => computeMessage(s.conditions),

          width: 45,
        },
        {
          label: "Cluster",
          value: (s: Source) => s.clusterName,
          width: 5,
        },
        {
          label: "URL",
          value: (s: Source) => {
            let text;
            let url;
            let link = false;

            switch (s.type) {
              case SourceRefSourceKind.GitRepository:
                text = (s as GitRepository).url;
                url = convertGitURLToGitProvider((s as GitRepository).url);
                link = true;
                break;
              case SourceRefSourceKind.Bucket:
                text = (s as Bucket).endpoint;
                break;
              case SourceRefSourceKind.HelmChart:
                text = `https://${(s as HelmChart).sourceRef?.name}`;
                url = (s as HelmChart).chart;
                link = true;
                break;
              case SourceRefSourceKind.HelmRepository:
                text = (s as HelmRepository).url;
                url = text;
                link = true;
                break;
            }

            return link ? (
              <Link newTab href={url}>
                {text}
              </Link>
            ) : (
              text
            );
          },
          width: 25,
        },
        {
          label: "Reference",
          value: (s: Source) => {
            const isGit = s.type === SourceRefSourceKind.GitRepository;
            const repo = s as GitRepository;
            const ref =
              repo?.reference?.branch ||
              repo?.reference?.commit ||
              repo?.reference?.tag ||
              repo?.reference?.semver;

            return isGit ? ref : "-";
          },
          width: 5,
        },
        {
          label: "Interval",
          value: (s: Source) => showInterval(s.interval),
          width: 5,
        },
        {
          label: "Last Updated",
          value: (s: Source) => (
            <Timestamp time={(s as GitRepository).lastUpdatedAt} />
          ),
          width: 5,
        },
      ]}
    />
  );
}

export default styled(SourcesTable).attrs({ className: SourcesTable.name })`
  table {
    table-layout: fixed;
  }
`;
