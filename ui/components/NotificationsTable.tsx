import * as React from "react";
import styled from "styled-components";
import { useFeatureFlags } from "../hooks/featureflags";
import { formatURL } from "../lib/nav";
import { Provider } from "../lib/objects";
import { V2Routes } from "../lib/types";
import { statusSortHelper } from "../lib/utils";
import DataTable, {
  Field,
  filterByStatusCallback,
  filterConfig,
} from "./DataTable";
import Flex from "./Flex";
import KubeStatusIndicator from "./KubeStatusIndicator";
import MessageBox from "./MessageBox";
import Link from "./Link";
import Spacer from "./Spacer";
import Text from "./Text";

type Props = {
  className?: string;
  rows: Provider[];
};

function NotificationsTable({ className, rows }: Props) {
  const { data: flagData } = useFeatureFlags();
  const flags = flagData?.flags || {};

  let initialFilterState = {
    ...filterConfig(rows, "provider"),
    ...filterConfig(rows, "channel"),
    ...filterConfig(rows, "namespace"),
    ...filterConfig(rows, "status", filterByStatusCallback),
  };
  if (flags.WEAVE_GITOPS_FEATURE_CLUSTER === "true") {
    initialFilterState = {
      ...initialFilterState,
      ...filterConfig(rows, "clusterName"),
    };
  }
  if (flags.WEAVE_GITOPS_FEATURE_TENANCY === "true") {
    initialFilterState = {
      ...initialFilterState,
      ...filterConfig(rows, "tenant"),
    };
  }

  const providerFields: Field[] = [
    {
      label: "Name",
      value: (p) => {
        return (
          <Link
            to={formatURL(V2Routes.Provider, {
              name: p.name,
              namespace: p.namespace,
              clusterName: p.clusterName,
            })}
          >
            {p.name}
          </Link>
        );
      },
      sortValue: ({ name }) => name,
      textSearchable: true,
      defaultSort: true,
    },
    {
      label: "Type",
      value: "provider",
    },
    {
      label: "Channel",
      value: "channel",
    },
    {
      label: "Namespace",
      value: "namespace",
    },
    {
      label: "Status",
      value: (p: Provider) =>
        p.conditions.length > 0 ? (
          <KubeStatusIndicator
            short
            conditions={p.conditions}
            suspended={p.suspended}
          />
        ) : null,
      sortValue: statusSortHelper,
    },
    ...(flags.WEAVE_GITOPS_FEATURE_CLUSTER === "true"
      ? [{ label: "Cluster", value: (obj) => obj.clusterName }]
      : []),
    ...(flags.WEAVE_GITOPS_FEATURE_TENANCY === "true"
      ? [{ label: "Tenant", value: "tenant" }]
      : []),
  ];

  if (!rows?.length)
    return (
      <Flex wide tall column align>
        <MessageBox>
          <Text size="large" semiBold>
            No notifications are currently configured
          </Text>
          <Spacer padding="medium" />
          <Text>
            Set up notifications to alert on changes via multiple different
            platforms such as Slack, Azure Event Hub, Grafana, OpsGenie and many
            more!
          </Text>
          <Spacer padding="xs" />
          <Text>
            To learn more about how to set up notifications,&nbsp;
            <Link href="https://fluxcd.io/flux/guides/notifications/" newTab>
              visit our documentation
            </Link>
          </Text>
        </MessageBox>
      </Flex>
    );

  return (
    <DataTable
      className={className}
      rows={rows}
      fields={providerFields}
      filters={initialFilterState}
    />
  );
}

export default styled(NotificationsTable).attrs({
  className: NotificationsTable.name,
})``;
