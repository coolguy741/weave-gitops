import * as React from "react";
import styled from "styled-components";
import Page from "../../components/Page";
import ProviderDetail from "../../components/ProviderDetail";
import { useGetObject } from "../../hooks/objects";
import { Kind } from "../../lib/api/core/types.pb";
import { Provider } from "../../lib/objects";

type Props = {
  className?: string;
  name?: string;
  namespace?: string;
  clusterName?: string;
};

function ProviderPage({ className, name, namespace, clusterName }: Props) {
  const { data, isLoading, error } = useGetObject<Provider>(
    name,
    namespace,
    Kind.Provider,
    clusterName
  );
  return (
    <Page className={className} loading={isLoading} error={error}>
      <ProviderDetail provider={data} />
    </Page>
  );
}

export default styled(ProviderPage).attrs({ className: ProviderPage.name })``;
