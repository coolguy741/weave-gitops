import * as React from "react";
import styled from "styled-components";
import { FluxObjectKind, HelmRelease } from "../lib/api/core/types.pb";
import { automationLastUpdated } from "../lib/utils";
import Alert from "./Alert";
import AutomationDetail from "./AutomationDetail";
import Interval from "./Interval";
import { routeTab } from "./KustomizationDetail";
import SourceLink from "./SourceLink";
import Timestamp from "./Timestamp";

type Props = {
  name: string;
  clusterName: string;
  helmRelease?: HelmRelease;
  className?: string;
  customTabs?: Array<routeTab>;
};

function helmChartLink(helmRelease: HelmRelease) {
  if (helmRelease?.helmChartName === "") {
    return (
      <SourceLink
        sourceRef={{
          kind: FluxObjectKind.KindHelmChart,
          name: helmRelease?.helmChart.chart,
        }}
        clusterName={helmRelease?.clusterName}
      />
    );
  }

  const [ns, name] = helmRelease?.helmChartName
    ? helmRelease.helmChartName.split("/")
    : ["", ""];

  return (
    <SourceLink
      sourceRef={{
        kind: FluxObjectKind.KindHelmChart,
        name: name,
        namespace: ns,
      }}
      clusterName={helmRelease.clusterName}
    />
  );
}

function HelmReleaseDetail({ helmRelease, className, customTabs }: Props) {
  return (
    <AutomationDetail
      className={className}
      automation={{ ...helmRelease, kind: FluxObjectKind.KindHelmRelease }}
      customTabs={customTabs}
      info={[
        ["Source", helmChartLink(helmRelease)],
        ["Chart", helmRelease?.helmChart.chart],
        ["Cluster", helmRelease?.clusterName],
        ["Interval", <Interval interval={helmRelease?.interval} />],
        [
          "Last Updated",
          <Timestamp time={automationLastUpdated(helmRelease)} />,
        ],
      ]}
    />
  );
}

export default styled(HelmReleaseDetail).attrs({
  className: HelmReleaseDetail.name,
})`
  width: 100%;

  ${Alert} {
    margin-bottom: 16px;
  }
`;
