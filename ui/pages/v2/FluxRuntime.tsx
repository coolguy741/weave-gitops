import * as React from "react";
import styled from "styled-components";
import Page from "../../components/Page";
import Spacer from "../../components/Spacer";
import FluxRuntimeComponent from "../../components/FluxRuntime";
import { useListFluxRuntimeObjects } from "../../hooks/flux";

type Props = {
  className?: string;
};

function FluxRuntime({ className }: Props) {
  const { data, isLoading, error } = useListFluxRuntimeObjects();

  return (
    <Page
      title="Flux Runtime"
      loading={isLoading}
      error={error}
      className={className}
    >
      <Spacer padding="xs" />
      <FluxRuntimeComponent deployments={data?.deployments} />
    </Page>
  );
}

export default styled(FluxRuntime).attrs({ className: FluxRuntime.name })``;
