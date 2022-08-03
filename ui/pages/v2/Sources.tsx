import * as React from "react";
import styled from "styled-components";
import Page from "../../components/Page";
import SourcesTable from "../../components/SourcesTable";
import { useListSources } from "../../hooks/sources";

type Props = {
  className?: string;
};

function SourcesList({ className }: Props) {
  const { data: sources, error, isLoading } = useListSources();
  return (
    <Page
      error={error || sources?.errors}
      loading={isLoading}
      className={className}
    >
      <SourcesTable sources={sources?.result} />
    </Page>
  );
}

export default styled(SourcesList).attrs({ className: SourcesList.name })`
  h3 {
    visibility: hidden;
  }
`;
