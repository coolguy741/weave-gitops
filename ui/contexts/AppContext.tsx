import * as React from "react";
import { useHistory } from "react-router-dom";
import { Applications } from "../lib/api/applications/applications.pb";
import { formatURL } from "../lib/nav";
import { FluxObject, FluxObjectNode } from "../lib/objects";
import {
  clearCallbackState,
  getCallbackState,
  getProviderToken,
  storeCallbackState,
  storeProviderToken,
} from "../lib/storage";
import { PageRoute, V2Routes } from "../lib/types";
import { notifySuccess } from "../lib/utils";

type AppState = {
  error: null | { fatal: boolean; message: string; detail?: string };
  nodeYaml: FluxObjectNode | null;
};

type AppSettings = {
  renderFooter: boolean;
};

export type LinkResolver = (incoming: string) => string;

export function defaultLinkResolver(incoming: string): string {
  return incoming;
}

export type AppContextType = {
  applicationsClient: typeof Applications;
  userConfigRepoName: string;
  doAsyncError: (message: string, detail: string) => void;
  clearAsyncError: () => void;
  setNodeYaml: (obj: FluxObject | FluxObjectNode) => void;
  appState: AppState;
  settings: AppSettings;
  linkResolver: LinkResolver;
  getProviderToken: typeof getProviderToken;
  storeProviderToken: typeof storeProviderToken;
  getCallbackState: typeof getCallbackState;
  storeCallbackState: typeof storeCallbackState;
  clearCallbackState: typeof clearCallbackState;
  navigate: {
    internal: (page: PageRoute | V2Routes, query?: any) => void;
    external: (url: string) => void;
  };
  notifySuccess: typeof notifySuccess;
  request: typeof window.fetch;
};

export const AppContext = React.createContext<AppContextType>(
  null as AppContextType
);

export interface AppProps {
  applicationsClient?: typeof Applications;
  linkResolver?: LinkResolver;
  children?: any;
  renderFooter?: boolean;
  notifySuccess?: typeof notifySuccess;
}

export default function AppContextProvider({
  applicationsClient,
  ...props
}: AppProps) {
  const history = useHistory();
  const [appState, setAppState] = React.useState({
    error: null,
    nodeYaml: null,
  });

  const clearAsyncError = () => {
    setAppState({
      ...appState,
      error: null,
    });
  };

  React.useEffect(() => {
    // clear the error state on navigation
    clearAsyncError();
  }, [window.location]);

  const doAsyncError = (message: string, detail: string) => {
    console.error(message);
    setAppState({
      ...appState,
      error: { message, detail },
    });
  };

  const setNodeYaml = (obj: FluxObject | FluxObjectNode) => {
    if (obj) setAppState({ ...appState, nodeYaml: obj });
    else setAppState({ ...appState, nodeYaml: null });
  };

  const value: AppContextType = {
    applicationsClient,
    userConfigRepoName: "wego-github-jlw-config-repo",
    doAsyncError,
    clearAsyncError,
    setNodeYaml,
    appState,
    linkResolver: props.linkResolver || defaultLinkResolver,
    getProviderToken,
    storeProviderToken,
    storeCallbackState,
    getCallbackState,
    clearCallbackState,
    notifySuccess: props.notifySuccess || notifySuccess,
    settings: {
      renderFooter: props.renderFooter,
    },
    navigate: {
      internal: (page: PageRoute, query?: any) => {
        const u = formatURL(page, query);

        history.push(u);
      },
      external: (url) => {
        if (process.env.NODE_ENV === "test") {
          return;
        }
        window.location.href = url;
      },
    },
    request: window.fetch,
  };

  return <AppContext.Provider {...props} value={value} />;
}
