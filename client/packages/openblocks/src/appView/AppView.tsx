import { RootComp } from "comps/comps/rootComp";
import { GetContainerParams, useCompInstance } from "comps/utils/useCompInstance";
import { createBrowserHistory } from "history";
import { CompActionTypes, deferAction, changeChildAction } from "openblocks-core";
import { HTMLAttributes, useEffect, useMemo, useRef } from "react";
import { Provider } from "react-redux";
import { Route, Router } from "react-router";
import { reduxStore } from "redux/store/store";
import { ExternalEditorContext } from "util/context/ExternalEditorContext";
import { Helmet } from "react-helmet";
import React from "react";

const browserHistory = createBrowserHistory();

export interface OpenblocksAppBootStrapOptions {
  /**
   * where to load application dsl and static assets
   */
  baseUrl?: string;
}

interface AppViewProps extends HTMLAttributes<HTMLDivElement> {
  appDsl?: any;
  moduleDSL?: any;
  moduleInputs?: any;
  baseUrl?: string;
  webUrl?: string;
}

/**
 * root component of application view
 */
export function AppView(props: AppViewProps) {
  const { appDsl, moduleDSL, moduleInputs, baseUrl, webUrl, ...divProps } = props;
  const params = useMemo<GetContainerParams<RootComp>>(() => {
    return {
      Comp: RootComp,
      initialValue: appDsl,
      reduceContext: {
        readOnly: true,
        moduleDSL: moduleDSL || {},
        applicationId: appDsl?.applicationId,
        parentApplicationPath: [],
      },
    };
  }, [appDsl, moduleDSL]);
  const [comp, container] = useCompInstance(params);
  const appId = appDsl?.applicationId;

  useEffect(() => {
    if (container && moduleInputs) {
      // @ts-ignore
      container.dispatch(
        changeChildAction('ui', 'comp', 'io', 'inputs', (inputs: any[]) => {
          const next = [...inputs];
          Object.keys(moduleInputs).forEach((key) => {
            // make user input comp disabled by default
            if (moduleInputs[key]?.disabled === undefined) {
              moduleInputs[key].disabled = true;
            }
            const idx = next.findIndex((i) => i.name === key);
            if (idx !== -1) {
              next[idx].defaultValue.comp = moduleInputs[key];
            }
          });
          return next;
        }),
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [moduleInputs]);

  // Get favicon from app settings if available
  const appSettingsFavicon = comp?.children.settings.children.favicon.getView();
  const faviconUrl = appSettingsFavicon || null;

  return (
    <div {...divProps}>
      {faviconUrl && (
        <Helmet>
          <link rel="icon" href={faviconUrl} />
        </Helmet>
      )}
      <Provider store={reduxStore}>
        <ExternalEditorContext.Provider
          value={{
            applicationId: appId,
            appType: 1,
            readOnly: true,
            hideHeader: true,
          }}
        >
          <Router history={browserHistory}>
            <Route path="/" render={() => comp?.getView()} />
          </Router>
        </ExternalEditorContext.Provider>
      </Provider>
    </div>
  );
}
