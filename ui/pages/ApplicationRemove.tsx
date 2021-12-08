import { CircularProgress } from "@material-ui/core";
import * as React from "react";
import styled from "styled-components";
import { GithubDeviceAuthModal } from "..";
import Alert from "../components/Alert";
import AuthAlert from "../components/AuthAlert";
import Button from "../components/Button";
import Flex from "../components/Flex";
import Page from "../components/Page";
import Spacer from "../components/Spacer";
import Text from "../components/Text";
import { AppContext } from "../contexts/AppContext";
import { useAppRemove } from "../hooks/applications";
import { GrpcErrorCodes } from "../lib/types";
import { poller } from "../lib/utils";

type Props = {
  className?: string;
  name: string;
};

const RepoRemoveStatus = ({ done }: { done: boolean }) =>
  done ? (
    <Alert
      severity="info"
      title="Removed from Git Repo"
      message="The application successfully removed from your git repository"
    />
  ) : null;

const ClusterRemoveStatus = ({ done }: { done: boolean }) =>
  done ? (
    <Alert
      severity="success"
      title="Removed from cluster"
      message="The application was removed from your cluster"
    />
  ) : (
    <Flex wide center align>
      <CircularProgress />
      <Spacer margin="small" />
      <div>Removing from cluster...</div>
    </Flex>
  );

const Prompt = ({ onRemove, name }: { name: string; onRemove: () => void }) => (
  <Flex column center>
    <Flex wide center>
      <Text size="large" bold>
        Are you sure you want to remove the application {name}?
      </Text>
    </Flex>
    <Flex wide center>
      <Spacer padding="small">
        Removing this application will remove any Kubernetes objects that were
        created by the application
      </Spacer>
    </Flex>
    <Flex wide center>
      <Spacer padding="small">
        <Button onClick={onRemove} variant="contained" color="secondary">
          Remove {name}
        </Button>
      </Spacer>
    </Flex>
  </Flex>
);

function ApplicationRemove({ className, name }: Props) {
  const { applicationsClient } = React.useContext(AppContext);

  const [repoRemoveRes, repoRemoving, error, remove] = useAppRemove();
  const [repoInfo, setRepoInfo] = React.useState({
    provider: null,
    repoName: null,
  });
  const [removedFromCluster, setRemovedFromCluster] = React.useState(false);
  const [authOpen, setAuthOpen] = React.useState(false);
  const [authSuccess, setAuthSuccess] = React.useState(false);
  const [appError, setAppError] = React.useState(null);

  React.useEffect(() => {
    (async () => {
      try {
        const {
          application: { url },
        } = await applicationsClient.GetApplication({
          name,
          namespace: "wego-system",
        });

        const { provider, name: repoName } =
          await applicationsClient.ParseRepoURL({ url });

        setRepoInfo({ provider, repoName });
      } catch (e) {
        setAppError(e.message);
      }
    })();
  }, [name]);

  React.useEffect(() => {
    if (repoRemoving || error) {
      return;
    }

    const poll = poller(() => {
      applicationsClient
        .GetApplication({ name, namespace: "wego-system" })
        .catch(() => {
          // Once we get a 404, the app is gone for good
          clearInterval(poll);
          setRemovedFromCluster(true);
        });
    }, 5000);

    return () => {
      clearInterval(poll);
    };
  }, [repoRemoveRes]);

  const handleRemoveClick = () => {
    remove(repoInfo.provider, { name, namespace: "wego-system" });
  };

  const handleAuthSuccess = () => {
    setAuthSuccess(true);
  };

  if (!repoInfo) {
    return <CircularProgress />;
  }

  if (appError) {
    return (
      <Page className={className}>
        <Alert severity="error" title="Error" message={appError} />
      </Page>
    );
  }
  return (
    <Page className={className}>
      {!authSuccess &&
        error &&
        (error.code === GrpcErrorCodes.Unauthenticated ? (
          <AuthAlert
            title="Error"
            provider={repoInfo.provider}
            onClick={() => setAuthOpen(true)}
          />
        ) : (
          <Alert title="Error" message={error?.message} />
        ))}
      {repoRemoving && (
        <Flex wide center align>
          <CircularProgress />
          <Spacer margin="small" />
          <div>Remove operation in progress...</div>
        </Flex>
      )}
      {!repoRemoveRes && !repoRemoving && !removedFromCluster && (
        <Prompt name={name} onRemove={handleRemoveClick} />
      )}
      {(repoRemoving || repoRemoveRes) && (
        <RepoRemoveStatus done={!repoRemoving} />
      )}
      <Spacer margin="small" />
      {repoRemoveRes && <ClusterRemoveStatus done={removedFromCluster} />}
      <GithubDeviceAuthModal
        onSuccess={handleAuthSuccess}
        onClose={() => setAuthOpen(false)}
        open={authOpen}
        repoName={name}
      />
    </Page>
  );
}

export default styled(ApplicationRemove).attrs({
  className: ApplicationRemove.name,
})``;
